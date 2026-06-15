package peer

import (
	"testing"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

func TestRevokedNodeNotActive(t *testing.T) {
	paths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	network, err := identity.GenerateNetwork(time.Now(), "")
	if err != nil {
		t.Fatalf("GenerateNetwork: %v", err)
	}
	worker, err := identity.GenerateIdentity(network.ID, identity.RoleWorker, time.Now())
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}
	store := NewStore(paths)
	record := Record{NodeID: worker.NodeID, NetworkID: worker.NetworkID, Role: worker.Role, PublicKeys: worker.PublicKeys, Status: StatusActive, AddedAt: time.Now()}
	if err := store.Add(record); err != nil {
		t.Fatalf("Add: %v", err)
	}
	active, err := store.ActivePeers()
	if err != nil {
		t.Fatalf("ActivePeers: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("active count = %d, want 1", len(active))
	}
	revokedAt := time.Now().UTC()
	if err := store.MarkRevoked(revocation.Record{NodeID: worker.NodeID, Role: worker.Role, RevokedAt: revokedAt, RevokedBy: identity.NodeID("node_master")}); err != nil {
		t.Fatalf("MarkRevoked: %v", err)
	}
	active, err = store.ActivePeers()
	if err != nil {
		t.Fatalf("ActivePeers after revoke: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active count after revoke = %d, want 0", len(active))
	}
}
