package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tailedbox/link/control"
	"github.com/tailedbox/link/store"
	"github.com/tailedbox/tailedbox/internal/buildinfo"
	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/ui"
)

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute(context.Background(), &stdout, &stderr, []string{"version"}, buildinfo.Info{Version: "test", Commit: "abc", Date: "now", GoVersion: "go-test"})
	if err != nil {
		t.Fatalf("version failed: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "tailedbox test") {
		t.Fatalf("version output missing version: %s", stdout.String())
	}
}

func TestInitWorkerAndStatusJSON(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	args := append(paths, "init", "--role", "worker")
	if err := Execute(context.Background(), &stdout, &stderr, args, buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	args = append(paths, "--json", "worker", "status")
	if err := Execute(context.Background(), &stdout, &stderr, args, buildinfo.Info{}); err != nil {
		t.Fatalf("worker status failed: %v\nstderr: %s", err, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload["role"] != "worker" {
		t.Fatalf("expected worker role, got %v", payload["role"])
	}
	if payload["identity_ready"] != true {
		t.Fatalf("expected ready identity, got %v", payload["identity_ready"])
	}
	if payload["agent_config_ready"] != true {
		t.Fatalf("expected ready agent config, got %v", payload["agent_config_ready"])
	}
	if _, ok := payload["known_nodes"]; ok {
		t.Fatalf("worker status should not expose cluster inventory: %s", stdout.String())
	}
}

func TestMasterStatusHasClusterAwareShape(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "master", "status"), buildinfo.Info{}); err != nil {
		t.Fatalf("master status failed: %v\nstderr: %s", err, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if _, ok := payload["current"]; !ok {
		t.Fatalf("master status missing current: %s", stdout.String())
	}
	if _, ok := payload["known_nodes"]; !ok {
		t.Fatalf("master status missing known_nodes: %s", stdout.String())
	}
	current, ok := payload["current"].(map[string]any)
	if !ok {
		t.Fatalf("master status current has unexpected shape: %#v", payload["current"])
	}
	if current["identity_ready"] != true {
		t.Fatalf("expected ready identity, got %v", current["identity_ready"])
	}
}

func TestDebugLogToggle(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "debug", "logs", "enable"), buildinfo.Info{}); err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "enabled") {
		t.Fatalf("expected enable output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "debug", "logs", "disable"), buildinfo.Info{}); err != nil {
		t.Fatalf("disable failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "disabled") {
		t.Fatalf("expected disable output, got %s", stdout.String())
	}
}

func TestHelpOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, []string{"--help"}, buildinfo.Info{}); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"version", "init", "master", "worker", "pg"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help missing %q:\n%s", want, output)
		}
	}
}

func TestNoArgsShowsRootHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, nil, buildinfo.Info{}); err != nil {
		t.Fatalf("no-args help failed: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"tailedbox", "Usage", "Core", "Future Surfaces"} {
		if !strings.Contains(output, want) {
			t.Fatalf("no-args help missing %q:\n%s", want, output)
		}
	}
}

func TestExecuteInteractiveFallsBackToHelpForNonTTY(t *testing.T) {
	var stdin, stdout, stderr bytes.Buffer
	if err := ExecuteInteractive(context.Background(), &stdin, &stdout, &stderr, nil, buildinfo.Info{}); err != nil {
		t.Fatalf("interactive fallback failed: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Usage") || !strings.Contains(output, "Future Surfaces") {
		t.Fatalf("expected help fallback for non-tty execution:\n%s", output)
	}
}

func TestInteractiveMenuItemsMapToCLICommands(t *testing.T) {
	a := &app{}
	for _, action := range ui.DefaultActions() {
		if action.Args == nil {
			continue
		}
		menuActionCommandPath(t, a, action)
	}
}

func TestInteractiveMenuCoversCLICommandLeaves(t *testing.T) {
	a := &app{}
	covered := make(map[string]bool)
	for _, action := range ui.DefaultActions() {
		if action.Args == nil {
			continue
		}
		covered[menuActionCommandPath(t, a, action)] = true
	}

	expected := make(map[string]bool)
	collectRunnableCommandPaths(rootCommand(), expected)

	var missing []string
	for path := range expected {
		if !covered[path] {
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("interactive menu is missing CLI command leaves:\n%s", strings.Join(missing, "\n"))
	}
}

func TestUninstallDryRunDoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	paths := testPathsInDir(dir)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "uninstall", "--dry-run"), buildinfo.Info{}); err != nil {
		t.Fatalf("uninstall dry run failed: %v\nstderr: %s", err, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"Dry run", "Would remove", "local node identity", "initialized again", filepath.Join(dir, "config.json"), filepath.Join(dir, "state"), filepath.Join(dir, "logs")} {
		if !strings.Contains(output, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, output)
		}
	}
	for _, path := range uninstallLocalFilesForTest(dir) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("dry run should not remove %s: %v", path, err)
		}
	}
}

func TestUninstallRequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	paths := testPathsInDir(dir)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "worker"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err := Execute(context.Background(), &stdout, &stderr, append(paths, "uninstall"), buildinfo.Info{})
	if err == nil {
		t.Fatal("expected uninstall without confirmation to fail")
	}
	if !strings.Contains(err.Error(), "--confirm-delete DELETE") {
		t.Fatalf("unexpected uninstall confirmation error: %v", err)
	}
	for _, path := range uninstallLocalFilesForTest(dir) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("unconfirmed uninstall should not remove %s: %v", path, err)
		}
	}
}

func TestUninstallRemovesLocalFiles(t *testing.T) {
	dir := t.TempDir()
	paths := testPathsInDir(dir)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "uninstall", "--confirm-delete", "DELETE"), buildinfo.Info{}); err != nil {
		t.Fatalf("uninstall failed: %v\nstderr: %s", err, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"Removed local Tailedbox files", "local node identity", "tailedbox init"} {
		if !strings.Contains(output, want) {
			t.Fatalf("uninstall output missing %q:\n%s", want, output)
		}
	}
	for _, path := range uninstallLocalFilesForTest(dir) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, stat err: %v", path, err)
		}
	}
}

func uninstallLocalFilesForTest(dir string) []string {
	return []string{
		filepath.Join(dir, "config.json"),
		filepath.Join(dir, "state"),
		filepath.Join(dir, "logs"),
		filepath.Join(dir, "state", "node.json"),
		filepath.Join(dir, "state", "node_identity_public.json"),
		filepath.Join(dir, "state", "secrets", "node_identity_ed25519.pem"),
		filepath.Join(dir, "state", "agent", "config.json"),
	}
}

func menuActionCommandPath(t *testing.T, a *app, action ui.Action) string {
	t.Helper()
	parsed, help, err := a.parseGlobalFlags(action.Args)
	if err != nil {
		t.Fatalf("menu action %q has invalid args %v: %v", action.Title, action.Args, err)
	}
	cmd, _ := rootCommand().find(parsed)
	if !help && cmd.run == nil {
		t.Fatalf("menu action %q does not resolve to a runnable CLI command: %v", action.Title, action.Args)
	}
	return cmd.path()
}

func collectRunnableCommandPaths(cmd *command, paths map[string]bool) {
	if cmd.run != nil {
		paths[cmd.path()] = true
	}
	for _, child := range cmd.children {
		collectRunnableCommandPaths(child, paths)
	}
}

func TestAgentStatusBeforeRun(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "worker"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "agent", "status"), buildinfo.Info{}); err != nil {
		t.Fatalf("agent status failed: %v\nstderr: %s", err, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid agent status json: %v\n%s", err, stdout.String())
	}
	if payload["state"] != "stopped" {
		t.Fatalf("expected stopped agent before run, got %s", stdout.String())
	}
	if payload["running"] != false {
		t.Fatalf("expected non-running agent before run, got %s", stdout.String())
	}
	if payload["memory_alloc_bytes"] != float64(0) {
		t.Fatalf("expected no memory heartbeat before run, got %s", stdout.String())
	}
}

func TestAgentRunWritesHeartbeat(t *testing.T) {
	dir := t.TempDir()
	paths := testPathsInDir(dir)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "worker"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		var runStdout, runStderr bytes.Buffer
		errCh <- Execute(ctx, &runStdout, &runStderr, append(paths, "agent", "run", "--heartbeat-interval", "20ms"), buildinfo.Info{})
	}()
	defer cancel()

	statusFile := filepath.Join(dir, "state", "agent", "status.json")
	var payload map[string]any
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(statusFile)
		if err == nil {
			payload = map[string]any{}
			if err := json.Unmarshal(data, &payload); err != nil {
				t.Fatalf("invalid agent status file json: %v\n%s", err, string(data))
			}
			if payload["state"] == "running" && payload["running"] == true {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if payload["state"] != "running" || payload["running"] != true {
		t.Fatalf("agent did not write running heartbeat: %#v", payload)
	}
	if alloc, ok := payload["memory_alloc_bytes"].(float64); !ok || alloc <= 0 {
		t.Fatalf("expected memory allocation in heartbeat, got %#v", payload["memory_alloc_bytes"])
	}
	resolvedPaths, err := config.ResolvePaths(config.LoadOptions{
		ConfigPath: filepath.Join(dir, "config.json"),
		StateDir:   filepath.Join(dir, "state"),
		LogDir:     filepath.Join(dir, "logs"),
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	controlSocket := control.SocketPath(store.Paths{StateDir: resolvedPaths.StateDir, AgentDir: resolvedPaths.AgentDir})
	socketInfo, err := os.Lstat(controlSocket)
	if err != nil {
		t.Fatalf("expected mesh control socket: %v", err)
	}
	if socketInfo.Mode()&os.ModeSocket == 0 {
		t.Fatalf("expected mesh control socket file, got mode %v", socketInfo.Mode())
	}
	meshStatusFile := filepath.Join(dir, "state", "mesh", "status.json")
	if _, err := os.Stat(meshStatusFile); err != nil {
		t.Fatalf("expected mesh status file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "diagnose"), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh diagnose failed while agent was running: %v\nstderr: %s", err, stderr.String())
	}
	var diagnose map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &diagnose); err != nil {
		t.Fatalf("invalid mesh diagnose json: %v\n%s", err, stdout.String())
	}
	if diagnose["agent_control_reachable"] != true {
		t.Fatalf("expected reachable agent control, got %s", stdout.String())
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("agent run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("agent run did not stop after context cancellation")
	}
}

func TestMeshStatusBeforeAgentRun(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "worker"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "status"), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh status failed: %v\nstderr: %s", err, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid mesh status json: %v\n%s", err, stdout.String())
	}
	if payload["enabled"] != false {
		t.Fatalf("expected disabled mesh by default, got %s", stdout.String())
	}
	if payload["state"] != "disabled" {
		t.Fatalf("expected disabled mesh state, got %s", stdout.String())
	}
}

func TestMeshEnableDisableUpdatesAgentConfig(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "enable", "--listen-udp-port", "42424"), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh enable failed: %v\nstderr: %s", err, stderr.String())
	}
	var enabled map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &enabled); err != nil {
		t.Fatalf("invalid mesh enable json: %v\n%s", err, stdout.String())
	}
	mesh, ok := enabled["mesh"].(map[string]any)
	if !ok {
		t.Fatalf("mesh enable output missing mesh config: %s", stdout.String())
	}
	if mesh["enabled"] != true || mesh["listen_udp_port"] != float64(42424) {
		t.Fatalf("unexpected enabled mesh config: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "status"), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh status failed: %v\nstderr: %s", err, stderr.String())
	}
	var status map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("invalid mesh status json: %v\n%s", err, stdout.String())
	}
	if status["enabled"] != true || status["state"] != "stopped" || status["listen_udp_port"] != float64(42424) {
		t.Fatalf("mesh status did not reflect enabled config: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "disable"), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh disable failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "status"), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh status after disable failed: %v\nstderr: %s", err, stderr.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("invalid mesh status json after disable: %v\n%s", err, stdout.String())
	}
	if status["enabled"] != false || status["state"] != "disabled" {
		t.Fatalf("mesh status did not reflect disabled config: %s", stdout.String())
	}
}

func TestMeshPeersReadsRuntimeStore(t *testing.T) {
	dir := t.TempDir()
	paths := testPathsInDir(dir)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	resolved, err := config.ResolvePaths(config.LoadOptions{
		ConfigPath: filepath.Join(dir, "config.json"),
		StateDir:   filepath.Join(dir, "state"),
		LogDir:     filepath.Join(dir, "logs"),
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if _, err := store.WritePeer(store.Paths{StateDir: resolved.StateDir, AgentDir: resolved.AgentDir}, store.PeerObservation{
		NodeID:              "node_worker",
		Role:                config.RoleWorker,
		IdentityFingerprint: "tbx1_worker",
		LastEndpoint:        "127.0.0.1:41677",
		LastSeenAt:          time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC),
		SessionState:        store.SessionStateConnected,
	}); err != nil {
		t.Fatalf("write peer: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "peers"), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh peers failed: %v\nstderr: %s", err, stderr.String())
	}
	var peers []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &peers); err != nil {
		t.Fatalf("invalid mesh peers json: %v\n%s", err, stdout.String())
	}
	if len(peers) != 1 || peers[0]["node_id"] != "node_worker" {
		t.Fatalf("unexpected peers: %s", stdout.String())
	}
}

func TestMeshPingRequiresRunningAgent(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "worker"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err := Execute(context.Background(), &stdout, &stderr, append(paths, "mesh", "ping", "node_master"), buildinfo.Info{})
	if err == nil {
		t.Fatal("expected mesh ping to require running agent")
	}
	if !strings.Contains(err.Error(), "local agent control socket") {
		t.Fatalf("unexpected mesh ping error: %v", err)
	}
}

func TestMeshPingOverRunningAgents(t *testing.T) {
	masterDir := t.TempDir()
	workerDir := t.TempDir()
	masterPaths := testPathsInDir(masterDir)
	workerPaths := testPathsInDir(workerDir)
	var stdout, stderr bytes.Buffer

	if err := Execute(context.Background(), &stdout, &stderr, append(masterPaths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init master failed: %v\nstderr: %s", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(workerPaths, "init", "--role", "worker"), buildinfo.Info{}); err != nil {
		t.Fatalf("init worker failed: %v\nstderr: %s", err, stderr.String())
	}
	masterCfg, err := config.Load(config.LoadOptions{
		ConfigPath: filepath.Join(masterDir, "config.json"),
		StateDir:   filepath.Join(masterDir, "state"),
		LogDir:     filepath.Join(masterDir, "logs"),
	})
	if err != nil {
		t.Fatalf("load master config: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(masterPaths, "master", "join-code", "create", "--role", "worker", "--ttl", "15m"), buildinfo.Info{}); err != nil {
		t.Fatalf("create join code failed: %v\nstderr: %s", err, stderr.String())
	}
	code := regexp.MustCompile(`tbxjc1\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`).FindString(stdout.String())
	if code == "" {
		t.Fatalf("join code missing from output:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(workerPaths, "worker", "join", "--code", code, "--master-state-dir", filepath.Join(masterDir, "state")), buildinfo.Info{}); err != nil {
		t.Fatalf("worker join failed: %v\nstderr: %s", err, stderr.String())
	}

	port := freeUDPPort(t)
	masterEndpoint := net.JoinHostPort("127.0.0.1", port)
	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(masterPaths, "mesh", "enable", "--listen-udp-port", port), buildinfo.Info{}); err != nil {
		t.Fatalf("enable master mesh failed: %v\nstderr: %s", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(workerPaths, "mesh", "enable", "--master-endpoint", masterEndpoint), buildinfo.Info{}); err != nil {
		t.Fatalf("enable worker mesh failed: %v\nstderr: %s", err, stderr.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	masterErr := runAgentForTest(ctx, masterPaths)
	workerErr := runAgentForTest(ctx, workerPaths)
	defer assertAgentStops(t, cancel, masterErr, workerErr)

	waitForMeshReady(t, masterPaths)
	waitForMeshReady(t, workerPaths)

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(workerPaths, "--json", "mesh", "ping", masterCfg.Node.ID), buildinfo.Info{}); err != nil {
		t.Fatalf("mesh ping failed: %v\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}
	var ping map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &ping); err != nil {
		t.Fatalf("invalid mesh ping json: %v\n%s", err, stdout.String())
	}
	if ping["success"] != true {
		t.Fatalf("expected successful mesh ping, got %s", stdout.String())
	}
}

func TestAgentInstallDryRunPrintsSystemdUnit(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	args := append(paths, "agent", "install", "--dry-run", "--binary", "/usr/local/bin/tailedbox")
	if err := Execute(context.Background(), &stdout, &stderr, args, buildinfo.Info{}); err != nil {
		t.Fatalf("agent install dry-run failed: %v\nstderr: %s", err, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"[Unit]",
		"ExecStart=\"/usr/local/bin/tailedbox\"",
		"agent\" \"run\"",
		"Restart=always",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("systemd unit output missing %q:\n%s", want, output)
		}
	}
}

func freeUDPPort(t *testing.T) string {
	t.Helper()
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve udp port: %v", err)
	}
	defer packetConn.Close()
	addr, ok := packetConn.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("unexpected udp addr: %T", packetConn.LocalAddr())
	}
	return strconv.Itoa(addr.Port)
}

func runAgentForTest(ctx context.Context, paths []string) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		var stdout, stderr bytes.Buffer
		errCh <- Execute(ctx, &stdout, &stderr, append(paths, "agent", "run", "--heartbeat-interval", "20ms"), buildinfo.Info{})
	}()
	return errCh
}

func waitForMeshReady(t *testing.T, paths []string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var stdout, stderr bytes.Buffer
		err := Execute(context.Background(), &stdout, &stderr, append(paths, "--json", "mesh", "diagnose"), buildinfo.Info{})
		if err == nil {
			var diagnose map[string]any
			if json.Unmarshal(stdout.Bytes(), &diagnose) == nil && diagnose["agent_control_reachable"] == true && diagnose["udp_transport_ready"] == true {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("mesh did not become ready")
}

func assertAgentStops(t *testing.T, cancel context.CancelFunc, channels ...<-chan error) {
	t.Helper()
	cancel()
	for _, ch := range channels {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("agent run returned error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("agent run did not stop")
		}
	}
}

func TestInitCreatesIdentityAndAgentState(t *testing.T) {
	dir := t.TempDir()
	paths := testPathsInDir(dir)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stateDir := filepath.Join(dir, "state")
	secretKey := filepath.Join(stateDir, "secrets", "node_identity_ed25519.pem")
	publicIdentity := filepath.Join(stateDir, "node_identity_public.json")
	nodeMetadata := filepath.Join(stateDir, "node.json")
	agentConfig := filepath.Join(stateDir, "agent", "config.json")

	for _, check := range []struct {
		path string
		mode os.FileMode
	}{
		{filepath.Join(dir, "config.json"), 0o600},
		{stateDir, 0o700},
		{filepath.Join(dir, "logs"), 0o700},
		{filepath.Join(stateDir, "secrets"), 0o700},
		{filepath.Join(stateDir, "agent"), 0o700},
		{filepath.Join(stateDir, "master"), 0o700},
		{secretKey, 0o600},
		{publicIdentity, 0o600},
		{nodeMetadata, 0o600},
		{agentConfig, 0o600},
	} {
		assertMode(t, check.path, check.mode)
	}

	privateKeyBefore, err := os.ReadFile(secretKey)
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("second init failed: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "already initialized") {
		t.Fatalf("expected idempotent init message, got %s", stdout.String())
	}
	privateKeyAfter, err := os.ReadFile(secretKey)
	if err != nil {
		t.Fatalf("read private key after second init: %v", err)
	}
	if string(privateKeyBefore) != string(privateKeyAfter) {
		t.Fatal("private key changed on idempotent init")
	}
}

func TestInitRefusesRoleChange(t *testing.T) {
	paths := testPaths(t)
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init master failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	err := Execute(context.Background(), &stdout, &stderr, append(paths, "init", "--role", "worker"), buildinfo.Info{})
	if err == nil {
		t.Fatal("expected role-change refusal")
	}
	if !strings.Contains(err.Error(), "refusing to change role") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJoinCodeCLIFlow(t *testing.T) {
	masterDir := t.TempDir()
	workerDir := t.TempDir()
	masterPaths := testPathsInDir(masterDir)
	workerPaths := testPathsInDir(workerDir)
	var stdout, stderr bytes.Buffer

	if err := Execute(context.Background(), &stdout, &stderr, append(masterPaths, "init", "--role", "master"), buildinfo.Info{}); err != nil {
		t.Fatalf("init master failed: %v\nstderr: %s", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(workerPaths, "init", "--role", "worker"), buildinfo.Info{}); err != nil {
		t.Fatalf("init worker failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(masterPaths, "master", "join-code", "create", "--role", "worker", "--ttl", "15m"), buildinfo.Info{}); err != nil {
		t.Fatalf("create join code failed: %v\nstderr: %s", err, stderr.String())
	}
	code := regexp.MustCompile(`tbxjc1\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`).FindString(stdout.String())
	if code == "" {
		t.Fatalf("join code missing from output:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	joinArgs := append(workerPaths, "worker", "join", "--code", code, "--master-state-dir", filepath.Join(masterDir, "state"))
	if err := Execute(context.Background(), &stdout, &stderr, joinArgs, buildinfo.Info{}); err != nil {
		t.Fatalf("worker join failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(workerPaths, "--json", "worker", "status"), buildinfo.Info{}); err != nil {
		t.Fatalf("worker status failed: %v\nstderr: %s", err, stderr.String())
	}
	var workerStatus map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &workerStatus); err != nil {
		t.Fatalf("invalid worker status json: %v\n%s", err, stdout.String())
	}
	if workerStatus["joined_to_master_cluster"] != true {
		t.Fatalf("expected joined worker status, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Execute(context.Background(), &stdout, &stderr, append(masterPaths, "--json", "master", "status"), buildinfo.Info{}); err != nil {
		t.Fatalf("master status failed: %v\nstderr: %s", err, stderr.String())
	}
	var masterStatus map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &masterStatus); err != nil {
		t.Fatalf("invalid master status json: %v\n%s", err, stdout.String())
	}
	nodes, ok := masterStatus["known_nodes"].([]any)
	if !ok || len(nodes) != 2 {
		t.Fatalf("expected master plus worker in known nodes, got %s", stdout.String())
	}
}

func testPaths(t *testing.T) []string {
	t.Helper()
	return testPathsInDir(t.TempDir())
}

func testPathsInDir(dir string) []string {
	return []string{
		"--config", filepath.Join(dir, "config.json"),
		"--state-dir", filepath.Join(dir, "state"),
		"--log-dir", filepath.Join(dir, "logs"),
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode for %s = %v, want %v", path, got, want)
	}
}
