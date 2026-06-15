package revocation

import (
	"testing"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func TestRevocationRecordCreation(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	record, err := NewRecord("node_worker", identity.RoleWorker, "node_master", Reason("retired"), now)
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	if record.NodeID != "node_worker" || record.Role != identity.RoleWorker || record.RevokedBy != "node_master" || record.Reason != "retired" {
		t.Fatalf("record mismatch: %#v", record)
	}
}

func TestRevocationStoreChecks(t *testing.T) {
	paths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	store := NewStore(paths).WithClock(func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) })
	record, err := store.Revoke("node_worker", identity.RoleWorker, "node_master", "removed")
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if record.RevokedAt.IsZero() || record.RevokedBy != "node_master" {
		t.Fatalf("record missing metadata: %#v", record)
	}
	revoked, err := store.IsRevoked("node_worker")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("node was not revoked")
	}
	revoked, err = store.IsRevoked("node_other")
	if err != nil {
		t.Fatalf("IsRevoked other: %v", err)
	}
	if revoked {
		t.Fatal("unexpected revoked state for other node")
	}
}
