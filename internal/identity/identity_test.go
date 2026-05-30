package identity

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

func TestEnsureCreatesDurableIdentity(t *testing.T) {
	cfg := testConfig(t)
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	first, err := Ensure(cfg, now)
	if err != nil {
		t.Fatalf("ensure identity: %v", err)
	}
	if !first.Created {
		t.Fatal("expected private key creation")
	}
	if cfg.Node.Identity.PublicKeyFingerprint == "" {
		t.Fatal("expected fingerprint in config")
	}
	assertMode(t, cfg.Paths.SecretsDir, secrets.PrivateDirMode)
	assertMode(t, cfg.Paths.IdentityPrivateKeyFile, secrets.PrivateFileMode)
	assertMode(t, cfg.Paths.IdentityPublicKeyFile, secrets.PrivateFileMode)

	privateKeyBytes, err := os.ReadFile(cfg.Paths.IdentityPrivateKeyFile)
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	second, err := Ensure(cfg, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("ensure identity again: %v", err)
	}
	if second.Created {
		t.Fatal("expected existing private key to be reused")
	}
	privateKeyBytesAgain, err := os.ReadFile(cfg.Paths.IdentityPrivateKeyFile)
	if err != nil {
		t.Fatalf("read private key again: %v", err)
	}
	if string(privateKeyBytes) != string(privateKeyBytesAgain) {
		t.Fatal("private key changed on idempotent ensure")
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	stateDir := filepath.Join(t.TempDir(), "state")
	paths, err := config.ResolvePaths(config.LoadOptions{
		ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		StateDir:   stateDir,
		LogDir:     filepath.Join(stateDir, "logs"),
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	return &config.Config{
		Version: 1,
		Paths:   paths,
		Node: config.NodeConfig{
			ID:   "node_test",
			Role: config.RoleWorker,
		},
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
