package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteStatusUsesPrivateMeshState(t *testing.T) {
	paths := testPaths(t)
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)

	changed, err := WriteStatus(paths, Status{
		NodeID:        "node_master",
		Role:          "master",
		Enabled:       true,
		State:         StateListening,
		Health:        HealthHealthy,
		ListenUDPPort: 41677,
		LastUpdatedAt: now,
		PeerCount:     1,
	})
	if err != nil {
		t.Fatalf("write status: %v", err)
	}
	if !changed {
		t.Fatal("expected status file creation")
	}

	runtimePaths, err := ResolvePaths(paths)
	if err != nil {
		t.Fatalf("resolve mesh paths: %v", err)
	}
	assertMode(t, runtimePaths.MeshDir, 0o700)
	assertMode(t, runtimePaths.PeersDir, 0o700)
	assertMode(t, runtimePaths.StatusFile, 0o600)

	status, err := ReadStatus(paths)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status.Version != 1 {
		t.Fatalf("version = %d, want 1", status.Version)
	}
	if status.NodeID != "node_master" || status.ListenUDPPort != 41677 {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestWritePeerListsSortedPrivatePeerFiles(t *testing.T) {
	paths := testPaths(t)
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	for _, peer := range []PeerObservation{
		{
			NodeID:              "node_z",
			Role:                "worker",
			IdentityFingerprint: "tbx1_z",
			LastEndpoint:        "203.0.113.10:41677",
			LastSeenAt:          now,
			SessionState:        SessionStateConnected,
		},
		{
			NodeID:              "node_a",
			Role:                "worker",
			IdentityFingerprint: "tbx1_a",
			LastEndpoint:        "203.0.113.11:41677",
			LastSeenAt:          now,
			SessionState:        SessionStateDegraded,
		},
	} {
		if _, err := WritePeer(paths, peer); err != nil {
			t.Fatalf("write peer %s: %v", peer.NodeID, err)
		}
	}

	runtimePaths, err := ResolvePaths(paths)
	if err != nil {
		t.Fatalf("resolve mesh paths: %v", err)
	}
	assertMode(t, filepath.Join(runtimePaths.PeersDir, "node_a.json"), 0o600)
	assertMode(t, filepath.Join(runtimePaths.PeersDir, "node_z.json"), 0o600)

	peers, err := ListPeers(paths)
	if err != nil {
		t.Fatalf("list peers: %v", err)
	}
	if len(peers) != 2 {
		t.Fatalf("peer count = %d, want 2", len(peers))
	}
	if peers[0].NodeID != "node_a" || peers[1].NodeID != "node_z" {
		t.Fatalf("peers not sorted: %#v", peers)
	}

	peer, err := ReadPeer(paths, "node_z")
	if err != nil {
		t.Fatalf("read peer: %v", err)
	}
	if peer.Version != 1 || peer.IdentityFingerprint != "tbx1_z" {
		t.Fatalf("unexpected peer: %#v", peer)
	}
}

func TestWritePeerRejectsPathTraversalNodeID(t *testing.T) {
	_, err := WritePeer(testPaths(t), PeerObservation{NodeID: "../escape"})
	if err == nil || !strings.Contains(err.Error(), "invalid mesh peer node id") {
		t.Fatalf("expected invalid node id error, got %v", err)
	}
}

func testPaths(t *testing.T) Paths {
	t.Helper()
	return Paths{StateDir: filepath.Join(t.TempDir(), "state")}
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
