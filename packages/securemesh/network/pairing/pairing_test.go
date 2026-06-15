package pairing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/join"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/network/tlsidentity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

func TestJoinPairsWorkerWithMaster(t *testing.T) {
	fixture := newFixture(t)
	code := fixture.createJoinCode(t, identity.RoleWorker)
	listener := fixture.listen(t)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- listener.Serve(ctx) }()

	result, err := Join(ctx, fixture.workerPaths, listener.Addr().String(), code, identity.RoleWorker, fixture.master.NodeID)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	if result.MasterNodeID != fixture.master.NodeID {
		t.Fatalf("master node = %q, want %q", result.MasterNodeID, fixture.master.NodeID)
	}
	if result.NetworkID != fixture.network.ID {
		t.Fatalf("network = %q, want %q", result.NetworkID, fixture.network.ID)
	}

	workerIdentity, err := identity.LoadIdentity(fixture.workerPaths)
	if err != nil {
		t.Fatalf("LoadIdentity worker: %v", err)
	}
	if workerIdentity.NodeID != result.LocalNodeID {
		t.Fatalf("worker node = %q, want %q", workerIdentity.NodeID, result.LocalNodeID)
	}
	masterPeer, err := peer.NewStore(fixture.workerPaths).Get(fixture.master.NodeID)
	if err != nil {
		t.Fatalf("worker missing master peer: %v", err)
	}
	if !masterPeer.Active() {
		t.Fatal("master peer is not active on worker")
	}
	workerPeer, err := peer.NewStore(fixture.masterPaths).Get(workerIdentity.NodeID)
	if err != nil {
		t.Fatalf("master missing worker peer: %v", err)
	}
	if !workerPeer.Active() {
		t.Fatal("worker peer is not active on master")
	}

	_, err = join.NewStore(fixture.masterPaths).ValidateAndConsume(join.ConsumeRequest{
		Code:         code,
		NetworkID:    fixture.network.ID,
		ExpectedRole: identity.RoleWorker,
		ConsumedBy:   workerIdentity.NodeID,
	})
	if !errors.Is(err, join.ErrCodeConsumed) {
		t.Fatalf("code reuse err = %v, want ErrCodeConsumed", err)
	}

	cancel()
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve: %v", err)
	}
}

func TestJoinRejectsWrongMasterNode(t *testing.T) {
	fixture := newFixture(t)
	code := fixture.createJoinCode(t, identity.RoleWorker)
	listener := fixture.listen(t)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _ = listener.Serve(ctx) }()

	other, err := identity.GenerateIdentity(fixture.network.ID, identity.RoleMaster, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateIdentity other: %v", err)
	}
	_, err = Join(ctx, fixture.workerPaths, listener.Addr().String(), code, identity.RoleWorker, other.NodeID)
	if !errors.Is(err, ErrMasterMismatch) {
		t.Fatalf("Join err = %v, want ErrMasterMismatch", err)
	}
}

func TestListenerRejectsRevokedJoiningNode(t *testing.T) {
	fixture := newFixture(t)
	worker, err := identity.GenerateIdentity(fixture.network.ID, identity.RoleWorker, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateIdentity worker: %v", err)
	}
	if _, err := revocation.NewStore(fixture.masterPaths).Revoke(worker.NodeID, worker.Role, fixture.master.NodeID, "test revoke"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	code := fixture.createJoinCode(t, identity.RoleWorker)
	listener := &Listener{paths: fixture.masterPaths, local: fixture.master}
	_, err = listener.accept(pairRequest{
		Version: messageVersion,
		Type:    messageTypePairRequest,
		Code:    code,
		Role:    identity.RoleWorker,
		Peer:    PublicPeerFromIdentity(worker),
		Time:    time.Now().UTC(),
	})
	if !errors.Is(err, tlsidentity.ErrRevokedPeer) {
		t.Fatalf("accept err = %v, want ErrRevokedPeer", err)
	}
}

func TestNormalizeEndpointDefaultPort(t *testing.T) {
	endpoint, err := NormalizeEndpoint("example.com")
	if err != nil {
		t.Fatalf("NormalizeEndpoint: %v", err)
	}
	if endpoint != "example.com:9444" {
		t.Fatalf("endpoint = %q, want example.com:9444", endpoint)
	}
}

type fixture struct {
	network     identity.Network
	master      identity.Identity
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
	network.CreatedBy = master.NodeID
	masterPaths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths master: %v", err)
	}
	workerPaths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths worker: %v", err)
	}
	if err := identity.SaveNetwork(masterPaths, network); err != nil {
		t.Fatalf("SaveNetwork: %v", err)
	}
	if err := identity.SaveIdentity(masterPaths, master); err != nil {
		t.Fatalf("SaveIdentity: %v", err)
	}
	return fixture{network: network, master: master, masterPaths: masterPaths, workerPaths: workerPaths}
}

func (f fixture) createJoinCode(t *testing.T, role identity.Role) string {
	t.Helper()
	code, _, err := join.NewStore(f.masterPaths).Create(join.CreateRequest{
		NetworkID:    f.network.ID,
		ExpectedRole: role,
		IssuedBy:     f.master.NodeID,
	})
	if err != nil {
		t.Fatalf("Create join code: %v", err)
	}
	return code
}

func (f fixture) listen(t *testing.T) *Listener {
	t.Helper()
	listener, err := NewServer(f.masterPaths).Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	return listener
}
