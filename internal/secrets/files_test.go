package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileNewUsesStrictPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "secrets")
	path := filepath.Join(dir, "secret.txt")
	created, err := WriteFileNew(path, []byte("secret"))
	if err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if !created {
		t.Fatal("expected file creation")
	}
	assertMode(t, dir, PrivateDirMode)
	assertMode(t, path, PrivateFileMode)
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
