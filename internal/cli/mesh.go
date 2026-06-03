package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/tailedbox/tailedbox/internal/agent"
	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/mesh/control"
	"github.com/tailedbox/tailedbox/internal/mesh/store"
)

func meshCommand() *command {
	mesh := &command{
		name:        "mesh",
		usage:       "tailedbox mesh <command> [flags]",
		summary:     "Mesh diagnostics and peer commands",
		description: "Inspect local mesh runtime state and ask the running agent for live mesh operations.",
	}
	attach(mesh,
		&command{
			name:        "enable",
			usage:       "tailedbox mesh enable [--listen-udp-port 41677] [--master-endpoint host:port]",
			summary:     "Enable mesh runtime",
			description: "Enable the mesh runtime in the local agent config. Workers can store a master UDP endpoint with --master-endpoint.",
			needsConfig: true,
			run:         runMeshEnable,
		},
		&command{
			name:        "disable",
			usage:       "tailedbox mesh disable",
			summary:     "Disable mesh runtime",
			description: "Disable the mesh runtime in the local agent config without changing node identity or enrollment state.",
			needsConfig: true,
			run:         runMeshDisable,
		},
		&command{
			name:        "status",
			usage:       "tailedbox mesh status [--json]",
			summary:     "Show mesh status",
			description: "Show local mesh runtime status from the running agent when available, with a private state-file fallback.",
			needsConfig: true,
			run:         runMeshStatus,
		},
		&command{
			name:        "peers",
			usage:       "tailedbox mesh peers [--json]",
			summary:     "List mesh peers",
			description: "List mesh peer observations from the running agent when available, with a private state-file fallback.",
			needsConfig: true,
			run:         runMeshPeers,
		},
		&command{
			name:        "ping",
			usage:       "tailedbox mesh ping <node-id>",
			summary:     "Ping a mesh peer",
			description: "Ask the running local agent to ping a mesh peer. UDP encrypted ping arrives in the next transport slice.",
			needsConfig: true,
			run:         runMeshPing,
		},
		&command{
			name:        "diagnose",
			usage:       "tailedbox mesh diagnose [--json]",
			summary:     "Diagnose mesh connectivity",
			description: "Diagnose local mesh readiness, agent control reachability, runtime state, and next transport prerequisites.",
			needsConfig: true,
			run:         runMeshDiagnose,
		},
	)
	return mesh
}

type meshConfigResult struct {
	Changed         bool             `json:"changed"`
	AgentConfigFile string           `json:"agent_config_file"`
	Mesh            agent.MeshConfig `json:"mesh"`
	MasterEndpoints []string         `json:"master_endpoints,omitempty"`
}

func runMeshEnable(_ context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("mesh enable", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	listenUDPPort := fs.Int("listen-udp-port", 0, "UDP port for mesh listeners; masters default to 41677")
	masterEndpoint := fs.String("master-endpoint", "", "Master mesh UDP endpoint for workers, formatted as host:port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	result, err := agent.UpdateMeshConfig(a.cfg, agent.MeshUpdateOptions{
		Enabled:       true,
		ListenUDPPort: *listenUDPPort,
		Now:           time.Now(),
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(*masterEndpoint) != "" {
		endpoint, err := normalizeEndpoint(*masterEndpoint)
		if err != nil {
			return err
		}
		if !containsString(a.cfg.Cluster.MasterEndpoints, endpoint) {
			a.cfg.Cluster.MasterEndpoints = append(a.cfg.Cluster.MasterEndpoints, endpoint)
			if err := a.saveConfig(); err != nil {
				return err
			}
		}
	}
	payload := meshConfigResult{
		Changed:         result.Changed,
		AgentConfigFile: result.Path,
		Mesh:            result.Config.Mesh,
		MasterEndpoints: append([]string(nil), a.cfg.Cluster.MasterEndpoints...),
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, payload)
	}
	writeMeshConfigResult(a.stdout, a.theme, "Mesh enabled.", payload)
	return nil
}

func runMeshDisable(_ context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("mesh disable", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	result, err := agent.UpdateMeshConfig(a.cfg, agent.MeshUpdateOptions{
		Enabled: false,
		Now:     time.Now(),
	})
	if err != nil {
		return err
	}
	payload := meshConfigResult{
		Changed:         result.Changed,
		AgentConfigFile: result.Path,
		Mesh:            result.Config.Mesh,
		MasterEndpoints: append([]string(nil), a.cfg.Cluster.MasterEndpoints...),
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, payload)
	}
	writeMeshConfigResult(a.stdout, a.theme, "Mesh disabled.", payload)
	return nil
}

func runMeshStatus(ctx context.Context, a *app, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected argument %q", args[0])
	}
	status, live, err := meshStatus(ctx, a)
	if err != nil {
		return err
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, status)
	}
	writeMeshStatus(a.stdout, a.theme, status, live)
	return nil
}

func runMeshPeers(ctx context.Context, a *app, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected argument %q", args[0])
	}
	peers, live, err := meshPeers(ctx, a)
	if err != nil {
		return err
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, peers)
	}
	writeMeshPeers(a.stdout, a.theme, peers, live)
	return nil
}

func runMeshPing(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("mesh ping", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("mesh ping requires a peer node id")
	}
	peerNodeID := strings.TrimSpace(fs.Arg(0))
	response, err := meshControl(ctx, a, control.Request{
		Operation:  control.OperationPing,
		PeerNodeID: peerNodeID,
	})
	if err != nil {
		return fmt.Errorf("mesh ping requires the local agent control socket: %w", err)
	}
	if !response.OK {
		if response.Ping != nil && a.jsonOutput {
			return writeJSON(a.stdout, response.Ping)
		}
		return errors.New(response.Error)
	}
	if response.Ping == nil {
		return errors.New("mesh agent returned an empty ping response")
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, response.Ping)
	}
	writeMeshPing(a.stdout, a.theme, *response.Ping)
	return nil
}

func runMeshDiagnose(ctx context.Context, a *app, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected argument %q", args[0])
	}
	diagnose, err := meshDiagnose(ctx, a)
	if err != nil {
		return err
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, diagnose)
	}
	writeMeshDiagnose(a.stdout, a.theme, diagnose)
	return nil
}

func meshStatus(ctx context.Context, a *app) (store.Status, bool, error) {
	response, err := meshControl(ctx, a, control.Request{Operation: control.OperationStatus})
	if err == nil && response.OK && response.Status != nil {
		return *response.Status, true, nil
	}
	status, readErr := store.ReadStatus(a.cfg.Paths)
	if readErr == nil {
		status.Message = appendMessage(status.Message, "local agent control socket is not reachable; showing stored mesh status")
		return status, false, nil
	}
	if !errors.Is(readErr, os.ErrNotExist) {
		return store.Status{}, false, readErr
	}
	return defaultMeshStatus(a.cfg, "mesh status has not been written by the agent yet"), false, nil
}

func meshPeers(ctx context.Context, a *app) ([]store.PeerObservation, bool, error) {
	response, err := meshControl(ctx, a, control.Request{Operation: control.OperationPeers})
	if err == nil && response.OK {
		return response.Peers, true, nil
	}
	peers, readErr := store.ListPeers(a.cfg.Paths)
	if readErr != nil {
		return nil, false, readErr
	}
	return peers, false, nil
}

func meshDiagnose(ctx context.Context, a *app) (control.DiagnoseResult, error) {
	response, err := meshControl(ctx, a, control.Request{Operation: control.OperationDiagnose})
	if err == nil && response.OK && response.Diagnose != nil {
		return *response.Diagnose, nil
	}
	return localMeshDiagnose(a)
}

func meshControl(ctx context.Context, a *app, request control.Request) (control.Response, error) {
	controlCtx, cancel := context.WithTimeout(ctx, control.DefaultTimeout)
	defer cancel()
	return control.RoundTrip(controlCtx, a.cfg.Paths, request)
}

func defaultMeshStatus(cfg *config.Config, message string) store.Status {
	meshConfig := agent.MeshConfig{}
	if value, err := agent.ReadConfig(cfg); err == nil {
		meshConfig = value.Mesh
	} else if cfg.Node.Role == config.RoleMaster {
		meshConfig.ListenUDPPort = agent.DefaultMasterMeshUDPPort
	}
	enabled := meshConfig.Enabled
	state := store.StateDisabled
	health := store.HealthHealthy
	if enabled {
		state = store.StateStopped
		health = store.HealthDegraded
	}
	return store.Status{
		Version:       1,
		NodeID:        cfg.Node.ID,
		Role:          cfg.Node.Role,
		Enabled:       enabled,
		State:         state,
		Health:        health,
		ListenUDPPort: meshConfig.ListenUDPPort,
		LastUpdatedAt: time.Now().UTC(),
		Message:       message,
	}
}

func localMeshDiagnose(a *app) (control.DiagnoseResult, error) {
	status, _, err := meshStatus(context.Background(), a)
	if err != nil {
		return control.DiagnoseResult{}, err
	}
	runtimePaths, err := store.ResolvePaths(a.cfg.Paths)
	if err != nil {
		return control.DiagnoseResult{}, err
	}
	messages := []string{"local agent control socket is not reachable"}
	if status.Message != "" {
		messages = append(messages, status.Message)
	}
	if status.Enabled {
		messages = append(messages, "UDP mesh transport is not implemented in this Part 7 slice")
	} else {
		messages = append(messages, "mesh is disabled in the agent config")
	}
	if a.cfg.Node.Role == config.RoleWorker && len(a.cfg.Cluster.MasterEndpoints) == 0 {
		messages = append(messages, "worker has no configured master endpoints")
	}
	return control.DiagnoseResult{
		Version:               1,
		AgentControlReachable: false,
		MeshEnabled:           status.Enabled,
		UDPTransportReady:     false,
		NodeID:                status.NodeID,
		Role:                  status.Role,
		State:                 status.State,
		Health:                status.Health,
		ListenUDPPort:         status.ListenUDPPort,
		BoundEndpoint:         status.BoundEndpoint,
		PeerCount:             status.PeerCount,
		EstablishedPeerCount:  status.EstablishedPeerCount,
		ControlSocket:         control.SocketPath(a.cfg.Paths),
		StatusFile:            runtimePaths.StatusFile,
		Messages:              messages,
	}, nil
}

func appendMessage(existing, message string) string {
	if strings.TrimSpace(existing) == "" {
		return message
	}
	return existing + "; " + message
}

func normalizeEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", fmt.Errorf("master endpoint must be formatted as host:port: %w", err)
	}
	if strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
		return "", errors.New("master endpoint host and port are required")
	}
	return net.JoinHostPort(host, port), nil
}

func containsString(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}
