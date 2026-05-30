package logging

import (
	"log/slog"
	"os"
	"path/filepath"
)

func NewFileLogger(path string, debugEnabled bool) (*slog.Logger, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, err
	}
	level := slog.LevelInfo
	if debugEnabled {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})
	logger := slog.New(RedactingHandler{Next: handler})
	return logger, file.Close, nil
}
