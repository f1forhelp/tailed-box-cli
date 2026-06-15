package tlstcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

func TestPingOverTLS(t *testing.T) {
	fixture := newFixture(t)
	fixture.trustBoth(t)
	listener := fixture.listenMaster(t)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- listener.Serve(ctx) }()

	result, err := Ping(ctx, fixture.workerPaths, listener.Addr().String())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if result.LocalNodeID != fixture.worker.NodeID {
		t.Fatalf("local node = %q, want %q", result.LocalNodeID, fixture.worker.NodeID)
	}
	if result.RemoteNodeID != fixture.master.NodeID {
		t.Fatalf("remote node = %q, want %q", result.RemoteNodeID, fixture.master.NodeID)
	}
	if result.NetworkID != fixture.network.ID {
		t.Fatalf("network = %q, want %q", result.NetworkID, fixture.network.ID)
	}
}

func TestPingRejectsUnknownPeer(t *testing.T) {
	fixture := newFixture(t)
	fixture.trustWorkerOnMaster(t)
	listener := fixture.listenMaster(t)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _ = listener.Serve(ctx) }()

	_, err := Ping(ctx, fixture.workerPaths, listener.Addr().String())
	if err == nil {
		t.Fatal("Ping succeeded without worker trusting master")
	}
}

func TestPingRejectsRevokedPeer(t *testing.T) {
	fixture := newFixture(t)
	fixture.trustBoth(t)
	if _, err := revocation.NewStore(fixture.masterPaths).Revoke(fixture.worker.NodeID, fixture.worker.Role, fixture.master.NodeID, "test revoke"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	listener := fixture.listenMaster(t)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _ = listener.Serve(ctx) }()

	_, err := Ping(ctx, fixture.workerPaths, listener.Addr().String())
	if err == nil {
		t.Fatal("Ping succeeded with revoked worker")
	}
}

func TestNormalizeEndpointDefaultPort(t *testing.T) {
	endpoint, err := NormalizeEndpoint("example.com")
	if err != nil {
		t.Fatalf("NormalizeEndpoint: %v", err)
	}
	if endpoint != "example.com:9443" {
		t.Fatalf("endpoint = %q, want example.com:9443", endpoint)
	}
}

func TestReadMessageRejectsOversizedFrame(t *testing.T) {
	_, err := readMessage(bytesWithLength(MaxFrameSize + 1))
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("readMessage err = %v, want ErrFrameTooLarge", err)
	}
}

type fixture struct {
	network     identity.Network
	master      identity.Identity
	worker      identity.Identity
	masterPaths config.Paths
	workerPaths config.Paths
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	network, err := identity.GenerateNetwork(now, "")
	if err != nil {
		t.Fatalf("GenerateNetwork: %v", err)
	}
	master, err := identity.GenerateIdentity(network.ID, identity.RoleMaster, now)
	if err != nil {
		t.Fatalf("GenerateIdentity master: %v", err)
	}
	worker, err := identity.GenerateIdentity(network.ID, identity.RoleWorker, now)
	if err != nil {
		t.Fatalf("GenerateIdentity worker: %v", err)
	}
	masterPaths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths master: %v", err)
	}
	workerPaths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths worker: %v", err)
	}
	for _, item := range []struct {
		paths    config.Paths
		identity identity.Identity
	}{
		{paths: masterPaths, identity: master},
		{paths: workerPaths, identity: worker},
	} {
		if err := identity.SaveNetwork(item.paths, network); err != nil {
			t.Fatalf("SaveNetwork: %v", err)
		}
		if err := identity.SaveIdentity(item.paths, item.identity); err != nil {
			t.Fatalf("SaveIdentity: %v", err)
		}
	}
	return fixture{network: network, master: master, worker: worker, masterPaths: masterPaths, workerPaths: workerPaths}
}

func (f fixture) listenMaster(t *testing.T) *Listener {
	t.Helper()
	listener, err := NewServer(f.masterPaths).Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	return listener
}

func (f fixture) trustBoth(t *testing.T) {
	t.Helper()
	f.trustWorkerOnMaster(t)
	f.trustMasterOnWorker(t)
}

func (f fixture) trustWorkerOnMaster(t *testing.T) {
	t.Helper()
	addPeer(t, f.masterPaths, f.worker)
}

func (f fixture) trustMasterOnWorker(t *testing.T) {
	t.Helper()
	addPeer(t, f.workerPaths, f.master)
}

func addPeer(t *testing.T, paths config.Paths, node identity.Identity) {
	t.Helper()
	record := peer.Record{
		NodeID:     node.NodeID,
		NetworkID:  node.NetworkID,
		Role:       node.Role,
		PublicKeys: node.PublicKeys,
		Status:     peer.StatusActive,
		AddedAt:    time.Now().UTC(),
	}
	if err := peer.NewStore(paths).Add(record); err != nil {
		t.Fatalf("Add peer: %v", err)
	}
}

func bytesWithLength(length uint32) *testReader {
	return &testReader{data: []byte{byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)}}
}

type testReader struct {
	data []byte
}

func (r *testReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, errors.New("empty")
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}
