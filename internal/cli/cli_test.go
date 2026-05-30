package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tailedbox/tailedbox/internal/buildinfo"
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
