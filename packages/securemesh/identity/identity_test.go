package identity

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
)

func TestRoleValidation(t *testing.T) {
	for _, value := range []string{"master", "worker", " MASTER "} {
		if _, err := ParseRole(value); err != nil {
			t.Fatalf("ParseRole(%q): %v", value, err)
		}
	}
	if _, err := ParseRole("admin"); err == nil {
		t.Fatal("ParseRole accepted invalid role")
	}
}

func TestGenerateNetworkAndIdentity(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	network, err := GenerateNetwork(now, "")
	if err != nil {
		t.Fatalf("GenerateNetwork: %v", err)
	}
	if !strings.HasPrefix(network.ID.String(), "net_") {
		t.Fatalf("network id = %q", network.ID)
	}
	identity, err := GenerateIdentity(network.ID, RoleMaster, now)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	if !strings.HasPrefix(identity.NodeID.String(), "node_") {
		t.Fatalf("node id = %q", identity.NodeID)
	}
	if identity.Role != RoleMaster {
		t.Fatalf("role = %q", identity.Role)
	}
	if err := identity.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestDeriveNodeIDStable(t *testing.T) {
	network, err := GenerateNetwork(time.Now(), "")
	if err != nil {
		t.Fatalf("GenerateNetwork: %v", err)
	}
	identity, err := GenerateIdentity(network.ID, RoleWorker, time.Now())
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	first, err := DeriveNodeID(identity.PublicKeys)
	if err != nil {
		t.Fatalf("DeriveNodeID first: %v", err)
	}
	second, err := DeriveNodeID(identity.PublicKeys)
	if err != nil {
		t.Fatalf("DeriveNodeID second: %v", err)
	}
	if first != second || first != identity.NodeID {
		t.Fatalf("node id derivation mismatch: %q %q %q", first, second, identity.NodeID)
	}
}

func TestIdentitySaveLoadRestartSafeAndPermissions(t *testing.T) {
	paths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	network, err := GenerateNetwork(time.Now(), "")
	if err != nil {
		t.Fatalf("GenerateNetwork: %v", err)
	}
	identity, err := GenerateIdentity(network.ID, RoleMaster, time.Now())
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	if err := SaveNetwork(paths, network); err != nil {
		t.Fatalf("SaveNetwork: %v", err)
	}
	if err := SaveIdentity(paths, identity); err != nil {
		t.Fatalf("SaveIdentity: %v", err)
	}
	loaded, err := LoadIdentity(paths)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if loaded.NodeID != identity.NodeID || loaded.NetworkID != identity.NetworkID || loaded.Role != identity.Role {
		t.Fatalf("loaded identity mismatch: %#v != %#v", loaded, identity)
	}
	loadedNetwork, err := LoadNetwork(paths)
	if err != nil {
		t.Fatalf("LoadNetwork: %v", err)
	}
	if loadedNetwork.ID != network.ID {
		t.Fatalf("loaded network id = %q, want %q", loadedNetwork.ID, network.ID)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(paths.IdentityPath())
		if err != nil {
			t.Fatalf("stat identity: %v", err)
		}
		if got := info.Mode().Perm(); got != config.FileMode {
			t.Fatalf("identity mode = %v, want %v", got, config.FileMode)
		}
	}
}
