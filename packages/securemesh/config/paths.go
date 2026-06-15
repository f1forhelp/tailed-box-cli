package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppDirName = "tailed-box-cli"

	DirMode  os.FileMode = 0o700
	FileMode os.FileMode = 0o600
)

var ErrInvalidRoot = errors.New("invalid config root")

type Paths struct {
	Root string
}

func DefaultRoot() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppDirName), nil
}

func NewPaths(root string) (Paths, error) {
	if strings.TrimSpace(root) == "" {
		return Paths{}, ErrInvalidRoot
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Paths{}, err
	}
	return Paths{Root: filepath.Clean(abs)}, nil
}

func DefaultPaths() (Paths, error) {
	root, err := DefaultRoot()
	if err != nil {
		return Paths{}, err
	}
	return NewPaths(root)
}

func (p Paths) Ensure() error {
	if err := os.MkdirAll(p.Root, DirMode); err != nil {
		return err
	}
	if err := os.Chmod(p.Root, DirMode); err != nil {
		return err
	}
	if err := os.MkdirAll(p.LocksDir(), DirMode); err != nil {
		return err
	}
	return os.Chmod(p.LocksDir(), DirMode)
}

func (p Paths) IdentityPath() string {
	return filepath.Join(p.Root, "identity.json")
}

func (p Paths) NetworkPath() string {
	return filepath.Join(p.Root, "network.json")
}

func (p Paths) JoinCodesPath() string {
	return filepath.Join(p.Root, "join-codes.json")
}

func (p Paths) PeersPath() string {
	return filepath.Join(p.Root, "peers.json")
}

func (p Paths) RevocationsPath() string {
	return filepath.Join(p.Root, "revocations.json")
}

func (p Paths) LocksDir() string {
	return filepath.Join(p.Root, "locks")
}

func (p Paths) LockPath(name string) string {
	clean := filepath.Base(strings.TrimSpace(name))
	if clean == "." || clean == string(filepath.Separator) || clean == "" {
		clean = "state"
	}
	return filepath.Join(p.LocksDir(), clean+".lock")
}
