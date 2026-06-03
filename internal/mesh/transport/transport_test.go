package transport_test

import (
	"context"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/enrollment"
	"github.com/tailedbox/tailedbox/internal/mesh/store"
	"github.com/tailedbox/tailedbox/internal/mesh/transport"
	"github.com/tailedbox/tailedbox/internal/nodeinit"
)

func TestEnrolledWorkerCanPingMasterOverUDP(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	master := initializedConfig(t, "master", config.RoleMaster, now)
	worker := initializedConfig(t, "worker", config.RoleWorker, now)

	created, err := enrollment.CreateJoinCode(master, enrollment.CreateJoinCodeOptions{
		AllowedRole: config.RoleWorker,
		TTL:         15 * time.Minute,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("create join code: %v", err)
	}
	if _, err := enrollment.Join(worker, enrollment.JoinOptions{
		ExpectedRole:   config.RoleWorker,
		RawCode:        created.JoinCode,
		MasterStateDir: master.Paths.StateDir,
		Now:            now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("join worker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	masterTransport, err := transport.Start(ctx, master, transport.Options{
		ListenHost:    "127.0.0.1",
		ListenUDPPort: 0,
		Now:           func() time.Time { return now.Add(2 * time.Minute) },
		Logger:        logger,
	})
	if err != nil {
		t.Fatalf("start master transport: %v", err)
	}
	defer masterTransport.Close()
	workerTransport, err := transport.Start(ctx, worker, transport.Options{
		ListenHost:    "127.0.0.1",
		ListenUDPPort: 0,
		Now:           func() time.Time { return now.Add(2 * time.Minute) },
		Logger:        logger,
	})
	if err != nil {
		t.Fatalf("start worker transport: %v", err)
	}
	defer workerTransport.Close()

	masterEndpoint := net.JoinHostPort("127.0.0.1", strconv.Itoa(masterTransport.BoundUDPPort()))
	if _, err := workerTransport.Ping(ctx, master.Node.ID, masterEndpoint); err != nil {
		t.Fatalf("worker ping master: %v", err)
	}

	workerPeer, err := store.ReadPeer(worker.Paths, master.Node.ID)
	if err != nil {
		t.Fatalf("worker peer observation missing: %v", err)
	}
	if workerPeer.SessionState != store.SessionStateConnected || workerPeer.LastEndpoint != masterEndpoint {
		t.Fatalf("unexpected worker peer observation: %#v", workerPeer)
	}
	masterPeer, err := store.ReadPeer(master.Paths, worker.Node.ID)
	if err != nil {
		t.Fatalf("master peer observation missing: %v", err)
	}
	if masterPeer.SessionState != store.SessionStateConnected || masterPeer.LastEndpoint == "" {
		t.Fatalf("unexpected master peer observation: %#v", masterPeer)
	}
}

func initializedConfig(t *testing.T, name, role string, now time.Time) *config.Config {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	cfg, err := config.Load(config.LoadOptions{
		ConfigPath: filepath.Join(root, "config.json"),
		StateDir:   filepath.Join(root, "state"),
		LogDir:     filepath.Join(root, "logs"),
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := config.MarkInitialized(cfg, role, now); err != nil {
		t.Fatalf("mark initialized: %v", err)
	}
	if _, err := nodeinit.Ensure(cfg, now); err != nil {
		t.Fatalf("node init: %v", err)
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfg
}
