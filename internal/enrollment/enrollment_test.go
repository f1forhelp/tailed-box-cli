package enrollment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/nodeinit"
)

func TestCreateJoinCodeAndJoinWorker(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	master := initializedConfig(t, "master", config.RoleMaster, now)
	worker := initializedConfig(t, "worker", config.RoleWorker, now)

	created, err := CreateJoinCode(master, CreateJoinCodeOptions{
		AllowedRole: config.RoleWorker,
		TTL:         15 * time.Minute,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("create join code: %v", err)
	}
	if !strings.HasPrefix(created.JoinCode, JoinCodePrefix+".") {
		t.Fatalf("unexpected join code format: %s", created.JoinCode)
	}
	assertTreeDoesNotContain(t, master.Paths.StateDir, created.JoinCode)

	joined, err := Join(worker, JoinOptions{
		ExpectedRole:   config.RoleWorker,
		RawCode:        created.JoinCode,
		MasterStateDir: master.Paths.StateDir,
		Now:            now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("join worker: %v", err)
	}
	if joined.ClusterID != master.Cluster.ID {
		t.Fatalf("cluster id = %s, want %s", joined.ClusterID, master.Cluster.ID)
	}
	if worker.Cluster.ID != master.Cluster.ID {
		t.Fatalf("worker config cluster id = %s, want %s", worker.Cluster.ID, master.Cluster.ID)
	}
	record, err := ReadJoinCodeRecord(master.Paths, created.CodeID)
	if err != nil {
		t.Fatalf("read join record: %v", err)
	}
	if record.State != JoinCodeStateUsed || record.UsedByNodeID != worker.Node.ID {
		t.Fatalf("join code not marked used: %#v", record)
	}
	if _, err := ReadTrustedNode(master.Paths, worker.Node.ID); err != nil {
		t.Fatalf("trusted node missing: %v", err)
	}
	if _, err := ReadJoinedCluster(worker.Paths.JoinedClusterFile); err != nil {
		t.Fatalf("joined cluster missing: %v", err)
	}
	assertTreeDoesNotContain(t, master.Paths.StateDir, created.JoinCode)

	anotherWorker := initializedConfig(t, "another-worker", config.RoleWorker, now)
	_, err = Join(anotherWorker, JoinOptions{
		ExpectedRole:   config.RoleWorker,
		RawCode:        created.JoinCode,
		MasterStateDir: master.Paths.StateDir,
		Now:            now.Add(2 * time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "already been used") {
		t.Fatalf("expected one-time-use refusal, got %v", err)
	}
}

func TestJoinCodeRoleScope(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	master := initializedConfig(t, "master", config.RoleMaster, now)
	worker := initializedConfig(t, "worker", config.RoleWorker, now)

	created, err := CreateJoinCode(master, CreateJoinCodeOptions{
		AllowedRole: config.RoleMaster,
		TTL:         15 * time.Minute,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("create join code: %v", err)
	}
	_, err = Join(worker, JoinOptions{
		ExpectedRole:   config.RoleWorker,
		RawCode:        created.JoinCode,
		MasterStateDir: master.Paths.StateDir,
		Now:            now.Add(time.Minute),
	})
	if err == nil || !strings.Contains(err.Error(), "scoped to master") {
		t.Fatalf("expected role-scope refusal, got %v", err)
	}
}

func TestExpiredJoinCode(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	master := initializedConfig(t, "master", config.RoleMaster, now)
	worker := initializedConfig(t, "worker", config.RoleWorker, now)

	created, err := CreateJoinCode(master, CreateJoinCodeOptions{
		AllowedRole: config.RoleWorker,
		TTL:         time.Second,
		Now:         now,
	})
	if err != nil {
		t.Fatalf("create join code: %v", err)
	}
	_, err = Join(worker, JoinOptions{
		ExpectedRole:   config.RoleWorker,
		RawCode:        created.JoinCode,
		MasterStateDir: master.Paths.StateDir,
		Now:            now.Add(2 * time.Second),
	})
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expiry refusal, got %v", err)
	}
	record, err := ReadJoinCodeRecord(master.Paths, created.CodeID)
	if err != nil {
		t.Fatalf("read join record: %v", err)
	}
	if record.State != JoinCodeStateExpired {
		t.Fatalf("expected expired state, got %s", record.State)
	}
}

func TestOnlyMastersCreateJoinCodes(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	worker := initializedConfig(t, "worker", config.RoleWorker, now)
	_, err := CreateJoinCode(worker, CreateJoinCodeOptions{
		AllowedRole: config.RoleWorker,
		TTL:         15 * time.Minute,
		Now:         now,
	})
	if err == nil || !strings.Contains(err.Error(), "only initialized master") {
		t.Fatalf("expected master-only refusal, got %v", err)
	}
}

func initializedConfig(t *testing.T, name, role string, now time.Time) *config.Config {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	paths, err := config.ResolvePaths(config.LoadOptions{
		ConfigPath: filepath.Join(root, "config.json"),
		StateDir:   filepath.Join(root, "state"),
		LogDir:     filepath.Join(root, "logs"),
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	cfg := config.Default(paths)
	if _, err := config.MarkInitialized(cfg, role, now); err != nil {
		t.Fatalf("mark initialized: %v", err)
	}
	if _, err := nodeinit.Ensure(cfg, now); err != nil {
		t.Fatalf("ensure node init: %v", err)
	}
	return cfg
}

func assertTreeDoesNotContain(t *testing.T, root, needle string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), needle) {
			t.Fatalf("raw join code leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk state tree: %v", err)
	}
}
