package actions

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func TestControlActionEquivalentCLI(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	now := func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	options := []Option{WithConfigRoot(root), WithClock(now)}

	networkResult, err := InitNetwork(ctx, options...)
	if err != nil {
		t.Fatalf("InitNetwork: %v", err)
	}
	if networkResult.EquivalentCLI != "infra network init" {
		t.Fatalf("network equivalent = %q", networkResult.EquivalentCLI)
	}
	identityResult, err := InitIdentity(ctx, secureidentity.RoleMaster, options...)
	if err != nil {
		t.Fatalf("InitIdentity: %v", err)
	}
	if identityResult.EquivalentCLI != "infra identity init --role master" {
		t.Fatalf("identity equivalent = %q", identityResult.EquivalentCLI)
	}
	joinResult, err := CreateJoinCode(ctx, secureidentity.RoleWorker, options...)
	if err != nil {
		t.Fatalf("CreateJoinCode: %v", err)
	}
	if joinResult.EquivalentCLI != "infra join-code create --role worker" {
		t.Fatalf("join equivalent = %q", joinResult.EquivalentCLI)
	}
	if joinResult.SecretValue == "" {
		t.Fatal("join code secret not returned to caller")
	}
}

func TestPeerExportAddAndImportNetwork(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	masterOptions := []Option{WithConfigRoot(t.TempDir()), WithClock(now)}
	workerOptions := []Option{WithConfigRoot(t.TempDir()), WithClock(now)}

	networkResult, err := InitNetwork(ctx, masterOptions...)
	if err != nil {
		t.Fatalf("InitNetwork: %v", err)
	}
	networkID := secureidentity.NetworkID(networkResult.Fields["network_id"])
	if _, err := InitIdentity(ctx, secureidentity.RoleMaster, masterOptions...); err != nil {
		t.Fatalf("InitIdentity master: %v", err)
	}
	if _, err := ImportNetwork(ctx, networkID, workerOptions...); err != nil {
		t.Fatalf("ImportNetwork: %v", err)
	}
	if _, err := InitIdentity(ctx, secureidentity.RoleWorker, workerOptions...); err != nil {
		t.Fatalf("InitIdentity worker: %v", err)
	}

	exportResult, err := ExportPeer(ctx, workerOptions...)
	if err != nil {
		t.Fatalf("ExportPeer: %v", err)
	}
	if !strings.Contains(exportResult.RawOutput, "\"role\": \"worker\"") {
		t.Fatalf("export output missing role: %s", exportResult.RawOutput)
	}
	exported, err := DecodePeerExport([]byte(exportResult.RawOutput))
	if err != nil {
		t.Fatalf("DecodePeerExport: %v", err)
	}
	addResult, err := AddPeer(ctx, exported, masterOptions...)
	if err != nil {
		t.Fatalf("AddPeer: %v", err)
	}
	if addResult.EquivalentCLI != "infra peer add --file <peer.json>" {
		t.Fatalf("add equivalent = %q", addResult.EquivalentCLI)
	}
	listResult, err := ListPeers(ctx, masterOptions...)
	if err != nil {
		t.Fatalf("ListPeers: %v", err)
	}
	if listResult.Fields["count"] != "1" {
		t.Fatalf("peer count = %q, want 1", listResult.Fields["count"])
	}
}

func TestPairingActionsJoinWorker(t *testing.T) {
	ctx := context.Background()
	masterOptions := []Option{WithConfigRoot(t.TempDir())}
	workerOptions := []Option{WithConfigRoot(t.TempDir())}

	if _, err := InitNetwork(ctx, masterOptions...); err != nil {
		t.Fatalf("InitNetwork: %v", err)
	}
	masterIdentity, err := InitIdentity(ctx, secureidentity.RoleMaster, masterOptions...)
	if err != nil {
		t.Fatalf("InitIdentity master: %v", err)
	}
	joinCode, err := CreateJoinCode(ctx, secureidentity.RoleWorker, masterOptions...)
	if err != nil {
		t.Fatalf("CreateJoinCode: %v", err)
	}
	listener, err := PreparePairingListener(ctx, "127.0.0.1:0", masterOptions...)
	if err != nil {
		t.Fatalf("PreparePairingListener: %v", err)
	}
	defer listener.Close()
	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	serveDone := make(chan error, 1)
	go func() { serveDone <- listener.Serve(serveCtx) }()

	result, err := JoinPairing(ctx, listener.Addr, joinCode.SecretValue, secureidentity.RoleWorker, secureidentity.NodeID(masterIdentity.Fields["node_id"]), workerOptions...)
	if err != nil {
		t.Fatalf("JoinPairing: %v", err)
	}
	expectedCLI := "infra pair join --endpoint " + listener.Addr + " --code <code> --role worker --master-node " + masterIdentity.Fields["node_id"]
	if result.EquivalentCLI != expectedCLI {
		t.Fatalf("join equivalent = %q", result.EquivalentCLI)
	}
	masterPeers, err := ListPeers(ctx, masterOptions...)
	if err != nil {
		t.Fatalf("ListPeers master: %v", err)
	}
	if masterPeers.Fields["count"] != "1" {
		t.Fatalf("master peer count = %q, want 1", masterPeers.Fields["count"])
	}
	workerPeers, err := ListPeers(ctx, workerOptions...)
	if err != nil {
		t.Fatalf("ListPeers worker: %v", err)
	}
	if workerPeers.Fields["count"] != "1" {
		t.Fatalf("worker peer count = %q, want 1", workerPeers.Fields["count"])
	}
	cancel()
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve: %v", err)
	}
}

func TestWorkerCannotCreateJoinCode(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	options := []Option{WithConfigRoot(root)}
	if _, err := InitNetwork(ctx, options...); err != nil {
		t.Fatalf("InitNetwork: %v", err)
	}
	if _, err := InitIdentity(ctx, secureidentity.RoleWorker, options...); err != nil {
		t.Fatalf("InitIdentity: %v", err)
	}
	_, err := CreateJoinCode(ctx, secureidentity.RoleWorker, options...)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("CreateJoinCode err = %v, want ErrUnauthorized", err)
	}
}
