package cli

import (
	"bytes"
	"context"
	"encoding/json"
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

func testPaths(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	return []string{
		"--config", filepath.Join(dir, "config.json"),
		"--state-dir", filepath.Join(dir, "state"),
		"--log-dir", filepath.Join(dir, "logs"),
	}
}
