package private

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DirMode  os.FileMode = 0o700
	FileMode os.FileMode = 0o600
)

func EnsureDir(path string) error {
	if path == "" {
		return errors.New("private directory path is empty")
	}
	if err := os.MkdirAll(path, DirMode); err != nil {
		return fmt.Errorf("create private directory %q: %w", path, err)
	}
	if err := os.Chmod(path, DirMode); err != nil {
		return fmt.Errorf("secure private directory %q: %w", path, err)
	}
	return nil
}

func WriteFileAtomic(path string, data []byte) (bool, error) {
	if path == "" {
		return false, errors.New("private file path is empty")
	}
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return false, err
	}
	current, err := os.ReadFile(path)
	if err == nil && bytes.Equal(current, data) {
		if chmodErr := os.Chmod(path, FileMode); chmodErr != nil {
			return false, fmt.Errorf("secure private file %q: %w", path, chmodErr)
		}
		return false, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("read private file %q: %w", path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".secureconn-*.tmp")
	if err != nil {
		return false, fmt.Errorf("create temp file for %q: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(FileMode); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("secure temp file for %q: %w", path, err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("write temp file for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close temp file for %q: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return false, fmt.Errorf("replace private file %q: %w", path, err)
	}
	if err := os.Chmod(path, FileMode); err != nil {
		return false, fmt.Errorf("secure private file %q: %w", path, err)
	}
	return true, nil
}

func WriteJSONAtomic(path string, value any) (bool, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal json for %q: %w", path, err)
	}
	data = append(data, '\n')
	return WriteFileAtomic(path, data)
}

func ReadJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read json file %q: %w", path, err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("parse json file %q: %w", path, err)
	}
	return nil
}
