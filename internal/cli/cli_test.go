package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tailedbox/tailedbox/internal/buildinfo"
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
		parsed, help, err := a.parseGlobalFlags(action.Args)
		if err != nil {
			t.Fatalf("menu action %q has invalid args %v: %v", action.Title, action.Args, err)
		}
		cmd, _ := rootCommand().find(parsed)
		if !help && cmd.run == nil {
			t.Fatalf("menu action %q does not resolve to a runnable CLI command: %v", action.Title, action.Args)
		}
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
