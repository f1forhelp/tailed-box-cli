package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPathsAndPermissions(t *testing.T) {
	paths, err := NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	if err := paths.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if filepath.Base(paths.IdentityPath()) != "identity.json" {
		t.Fatalf("identity path = %s", paths.IdentityPath())
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(paths.Root)
		if err != nil {
			t.Fatalf("stat root: %v", err)
		}
		if got := info.Mode().Perm(); got != DirMode {
			t.Fatalf("root mode = %v, want %v", got, DirMode)
		}
	}
}

func TestAtomicWriteFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := AtomicWriteFile(path, []byte("{}\n"), FileMode); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "{}\n" {
		t.Fatalf("data = %q", data)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat file: %v", err)
		}
		if got := info.Mode().Perm(); got != FileMode {
			t.Fatalf("file mode = %v, want %v", got, FileMode)
		}
	}
}

func TestDirLock(t *testing.T) {
	paths, err := NewPaths(t.TempDir())
	if err != nil {
		t.Fatalf("NewPaths: %v", err)
	}
	lock, err := AcquireLock(paths.LockPath("join-codes"))
	if err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}
	if _, err := AcquireLock(paths.LockPath("join-codes")); err == nil {
		t.Fatal("second AcquireLock succeeded")
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if lock, err = AcquireLock(paths.LockPath("join-codes")); err != nil {
		t.Fatalf("AcquireLock after release: %v", err)
	}
	_ = lock.Release()
}
