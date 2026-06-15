package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func AtomicWriteFile(path string, data []byte, mode os.FileMode) error {
	if mode == 0 {
		mode = FileMode
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, DirMode); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	removeTmp = false
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return syncDir(dir)
}

func SaveJSON(path string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return AtomicWriteFile(path, data, mode)
}

func LoadJSON(path string, value any) error {
	data, err := ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
