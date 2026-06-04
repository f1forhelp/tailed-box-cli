package transport_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/tailedbox/secureconn/identity"
	"github.com/tailedbox/secureconn/store"
	"github.com/tailedbox/secureconn/transport"
)

func TestEnrolledWorkerCanPingMasterOverUDP(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	clusterID := "cluster_test"
	master := testNode(t, "node_master", "master", clusterID, now)
	worker := testNode(t, "node_worker", "worker", clusterID, now)
	masterObserver := newMemoryObserver()
	workerObserver := newMemoryObserver()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	masterTransport, err := transport.Start(ctx, master, transport.Options{
		ListenHost:     "127.0.0.1",
		ListenUDPPort:  0,
		Now:            func() time.Time { return now.Add(2 * time.Minute) },
		TrustValidator: trustMap{worker.NodeID: peerFromNode(worker)},
		PeerObserver:   masterObserver,
	})
	if err != nil {
		t.Fatalf("start master transport: %v", err)
	}
	defer masterTransport.Close()
	workerTransport, err := transport.Start(ctx, worker, transport.Options{
		ListenHost:     "127.0.0.1",
		ListenUDPPort:  0,
		Now:            func() time.Time { return now.Add(2 * time.Minute) },
		TrustValidator: trustMap{master.NodeID: peerFromNode(master)},
		PeerObserver:   workerObserver,
	})
	if err != nil {
		t.Fatalf("start worker transport: %v", err)
	}
	defer workerTransport.Close()

	masterEndpoint := net.JoinHostPort("127.0.0.1", strconv.Itoa(masterTransport.BoundUDPPort()))
	if _, err := workerTransport.Ping(ctx, master.NodeID, masterEndpoint); err != nil {
		t.Fatalf("worker ping master: %v", err)
	}

	workerPeer := workerObserver.peer(t, master.NodeID)
	if workerPeer.SessionState != store.SessionStateConnected || workerPeer.LastEndpoint != masterEndpoint {
		t.Fatalf("unexpected worker peer observation: %#v", workerPeer)
	}
	masterPeer := masterObserver.peer(t, worker.NodeID)
	if masterPeer.SessionState != store.SessionStateConnected || masterPeer.LastEndpoint == "" {
		t.Fatalf("unexpected master peer observation: %#v", masterPeer)
	}
}

func testNode(t *testing.T, nodeID, role, clusterID string, now time.Time) transport.LocalNode {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	publicIdentity := identity.PublicIdentity{
		Version:              1,
		NodeID:               nodeID,
		Algorithm:            identity.AlgorithmEd25519,
		PublicKey:            base64.RawStdEncoding.EncodeToString(publicKey),
		PublicKeyFingerprint: identity.Fingerprint(publicKey),
		CreatedAt:            now,
	}
	return transport.LocalNode{
		NodeID:          nodeID,
		Role:            role,
		ClusterID:       clusterID,
		PublicIdentity:  publicIdentity,
		PrivateIdentity: privateKey,
	}
}

func peerFromNode(node transport.LocalNode) transport.Peer {
	return transport.Peer{
		NodeID:              node.NodeID,
		Role:                node.Role,
		ClusterID:           node.ClusterID,
		IdentityFingerprint: node.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:      node.PublicIdentity,
	}
}

type trustMap map[string]transport.Peer

func (m trustMap) ValidateInitiator(peer transport.Peer) error {
	return m.validate(peer)
}

func (m trustMap) ValidateResponder(peer transport.Peer) error {
	return m.validate(peer)
}

func (m trustMap) validate(peer transport.Peer) error {
	trusted, ok := m[peer.NodeID]
	if !ok {
		return errors.New("peer is not trusted")
	}
	if trusted.Role != peer.Role || trusted.ClusterID != peer.ClusterID || trusted.IdentityFingerprint != peer.IdentityFingerprint {
		return errors.New("peer identity does not match trust record")
	}
	if trusted.PublicIdentity.PublicKey != peer.PublicIdentity.PublicKey {
		return errors.New("peer public key does not match trust record")
	}
	return nil
}

type memoryObserver struct {
	mu    sync.Mutex
	peers map[string]store.PeerObservation
}

func newMemoryObserver() *memoryObserver {
	return &memoryObserver{peers: make(map[string]store.PeerObservation)}
}

func (o *memoryObserver) ObservePeer(peer store.PeerObservation) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.peers[peer.NodeID] = peer
	return nil
}

func (o *memoryObserver) peer(t *testing.T, nodeID string) store.PeerObservation {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	peer, ok := o.peers[nodeID]
	if !ok {
		t.Fatalf("peer observation for %s missing", nodeID)
	}
	return peer
}
