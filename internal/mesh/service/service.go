package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/mesh/control"
	"github.com/tailedbox/tailedbox/internal/mesh/store"
	"github.com/tailedbox/tailedbox/internal/mesh/transport"
)

type MeshConfig struct {
	Enabled       bool
	Provider      string
	ListenUDPPort int
}

type Options struct {
	Logger *slog.Logger
	Now    func() time.Time
}

type Service struct {
	cfg         *config.Config
	meshConfig  MeshConfig
	listener    net.Listener
	controlPath string
	startedAt   time.Time
	logger      *slog.Logger
	now         func() time.Time
	transport   *transport.Transport
}

func Start(ctx context.Context, cfg *config.Config, meshConfig MeshConfig, opts Options) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if cfg.Node.ID == "" || cfg.Node.Role == "" {
		return nil, errors.New("node must be initialized before starting the mesh service")
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	listener, controlPath, err := control.Listen(cfg.Paths)
	if err != nil {
		return nil, err
	}
	service := &Service{
		cfg:         cfg,
		meshConfig:  meshConfig,
		listener:    listener,
		controlPath: controlPath,
		startedAt:   now().UTC(),
		logger:      opts.Logger,
		now:         now,
	}
	if meshConfig.Enabled {
		udpTransport, err := transport.Start(ctx, cfg, transport.Options{
			ListenUDPPort: meshConfig.ListenUDPPort,
			Logger:        opts.Logger,
			Now:           opts.Now,
		})
		if err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("start mesh UDP transport: %w", err)
		}
		service.transport = udpTransport
	}
	if err := service.writeStatus(); err != nil {
		_ = listener.Close()
		if service.transport != nil {
			_ = service.transport.Close()
		}
		return nil, err
	}
	go control.Serve(ctx, listener, service.handle)
	if opts.Logger != nil {
		opts.Logger.InfoContext(ctx, "mesh control socket started", "path", controlPath, "enabled", meshConfig.Enabled)
	}
	return service, nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	err := s.listener.Close()
	if s.transport != nil {
		if transportErr := s.transport.Close(); err == nil {
			err = transportErr
		}
	}
	_ = os.Remove(s.controlPath)
	return err
}

func (s *Service) RefreshStatus() error {
	if s == nil {
		return nil
	}
	return s.writeStatus()
}

func (s *Service) handle(ctx context.Context, request control.Request) control.Response {
	switch request.Operation {
	case control.OperationStatus:
		status, err := store.ReadStatus(s.cfg.Paths)
		if err != nil {
			return control.ErrorResponse(err)
		}
		response := control.OKResponse()
		response.Status = &status
		return response
	case control.OperationPeers:
		peers, err := store.ListPeers(s.cfg.Paths)
		if err != nil {
			return control.ErrorResponse(err)
		}
		response := control.OKResponse()
		response.Peers = peers
		return response
	case control.OperationPing:
		if request.PeerNodeID == "" {
			return control.ErrorResponse(errors.New("peer node id is required"))
		}
		if s.transport == nil {
			err := errors.New("mesh UDP transport is not running; run tailedbox mesh enable and restart the agent")
			response := control.ErrorResponse(err)
			response.Ping = &control.PingResult{
				Version:               1,
				PeerNodeID:            request.PeerNodeID,
				Success:               false,
				Message:               err.Error(),
				AgentControlReachable: true,
			}
			return response
		}
		endpoint, err := s.resolvePeerEndpoint(request.PeerNodeID)
		if err != nil {
			response := control.ErrorResponse(err)
			response.Ping = &control.PingResult{
				Version:               1,
				PeerNodeID:            request.PeerNodeID,
				Success:               false,
				Message:               err.Error(),
				AgentControlReachable: true,
			}
			return response
		}
		latency, err := s.transport.Ping(ctx, request.PeerNodeID, endpoint)
		if err != nil {
			response := control.ErrorResponse(err)
			response.Ping = &control.PingResult{
				Version:               1,
				PeerNodeID:            request.PeerNodeID,
				Success:               false,
				Message:               err.Error(),
				AgentControlReachable: true,
			}
			return response
		}
		response := control.OKResponse()
		response.Ping = &control.PingResult{
			Version:               1,
			PeerNodeID:            request.PeerNodeID,
			Success:               true,
			Message:               "mesh ping succeeded",
			LatencyMilliseconds:   latency.Milliseconds(),
			AgentControlReachable: true,
		}
		return response
	case control.OperationDiagnose:
		diagnose, err := s.diagnose(true)
		if err != nil {
			return control.ErrorResponse(err)
		}
		response := control.OKResponse()
		response.Diagnose = &diagnose
		return response
	default:
		return control.ErrorResponse(fmt.Errorf("unsupported mesh control operation %q", request.Operation))
	}
}

func (s *Service) writeStatus() error {
	status, err := s.status()
	if err != nil {
		return err
	}
	_, err = store.WriteStatus(s.cfg.Paths, status)
	return err
}

func (s *Service) status() (store.Status, error) {
	peers, err := store.ListPeers(s.cfg.Paths)
	if err != nil {
		return store.Status{}, err
	}
	now := s.now().UTC()
	status := store.Status{
		Version:       1,
		NodeID:        s.cfg.Node.ID,
		Role:          s.cfg.Node.Role,
		Enabled:       s.meshConfig.Enabled,
		ListenUDPPort: s.meshConfig.ListenUDPPort,
		StartedAt:     s.startedAt,
		LastUpdatedAt: now,
		PeerCount:     len(peers),
	}
	for _, peer := range peers {
		if peer.SessionState == store.SessionStateConnected {
			status.EstablishedPeerCount++
		}
	}
	if !s.meshConfig.Enabled {
		status.State = store.StateDisabled
		status.Health = store.HealthHealthy
		status.Message = "mesh is disabled in the agent config"
		return status, nil
	}
	if s.transport == nil {
		status.State = store.StateStopped
		status.Health = store.HealthDegraded
		status.Message = "mesh UDP transport is not running"
		return status, nil
	}
	status.State = store.StateListening
	status.Health = store.HealthHealthy
	status.BoundEndpoint = s.transport.BoundEndpoint()
	status.ListenUDPPort = s.transport.BoundUDPPort()
	status.Message = "mesh UDP transport listening"
	return status, nil
}

func (s *Service) diagnose(agentReachable bool) (control.DiagnoseResult, error) {
	status, err := s.status()
	if err != nil {
		return control.DiagnoseResult{}, err
	}
	runtimePaths, err := store.ResolvePaths(s.cfg.Paths)
	if err != nil {
		return control.DiagnoseResult{}, err
	}
	messages := []string{}
	if !s.meshConfig.Enabled {
		messages = append(messages, "mesh is disabled in the agent config")
	} else if s.transport == nil {
		messages = append(messages, "mesh UDP transport is not running")
	} else {
		messages = append(messages, "mesh UDP transport is listening")
	}
	if s.cfg.Node.Role == config.RoleMaster && s.meshConfig.ListenUDPPort == 0 {
		messages = append(messages, "master mesh UDP port is not configured")
	}
	if s.cfg.Node.Role == config.RoleWorker && len(s.cfg.Cluster.MasterEndpoints) == 0 {
		messages = append(messages, "worker has no configured master endpoints")
	}
	return control.DiagnoseResult{
		Version:               1,
		AgentControlReachable: agentReachable,
		MeshEnabled:           s.meshConfig.Enabled,
		UDPTransportReady:     s.transport != nil,
		NodeID:                status.NodeID,
		Role:                  status.Role,
		State:                 status.State,
		Health:                status.Health,
		ListenUDPPort:         status.ListenUDPPort,
		BoundEndpoint:         status.BoundEndpoint,
		PeerCount:             status.PeerCount,
		EstablishedPeerCount:  status.EstablishedPeerCount,
		ControlSocket:         control.SocketPath(s.cfg.Paths),
		StatusFile:            runtimePaths.StatusFile,
		Messages:              messages,
	}, nil
}

func (s *Service) resolvePeerEndpoint(peerNodeID string) (string, error) {
	if s.cfg.Node.Role == config.RoleWorker {
		if len(s.cfg.Cluster.MasterEndpoints) == 0 {
			return "", errors.New("worker has no configured master endpoints; run tailedbox mesh enable --master-endpoint <host:port>")
		}
		return s.cfg.Cluster.MasterEndpoints[0], nil
	}
	peer, err := store.ReadPeer(s.cfg.Paths, peerNodeID)
	if err == nil && peer.LastEndpoint != "" {
		return peer.LastEndpoint, nil
	}
	if s.cfg.Node.Role == config.RoleMaster {
		return "", errors.New("master has no observed endpoint for this peer yet; have the peer ping the master first")
	}
	return "", errors.New("mesh peer endpoint is not configured")
}
