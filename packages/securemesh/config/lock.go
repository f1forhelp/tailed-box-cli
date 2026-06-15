package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrLockHeld = errors.New("lock is already held")

type DirLock struct {
	path string
	held bool
}

func AcquireLock(path string) (*DirLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), DirMode); err != nil {
		return nil, err
	}
	if err := os.Mkdir(path, DirMode); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrLockHeld, path)
		}
		return nil, err
	}
	return &DirLock{path: path, held: true}, nil
}

func (l *DirLock) Release() error {
	if l == nil || !l.held {
		return nil
	}
	if err := os.Remove(l.path); err != nil {
		return err
	}
	l.held = false
	return nil
}

func (l *DirLock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}
