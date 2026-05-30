package logging

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

func PrintLastLines(w io.Writer, path string, limit int) error {
	lines, err := LastLines(path, limit)
	if err != nil {
		return err
	}
	for _, line := range lines {
		fmt.Fprintln(w, Redact(line))
	}
	return nil
}

func LastLines(path string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0, limit)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(lines) == limit {
			copy(lines, lines[1:])
			lines[len(lines)-1] = line
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func Follow(ctx context.Context, w io.Writer, path string, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	var offset int64
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		next, err := printSince(w, path, offset)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		offset = next

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func printSince(w io.Writer, path string, offset int64) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return offset, err
	}
	defer file.Close()

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Fprintln(w, Redact(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return offset, err
	}
	next, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return offset, err
	}
	return next, nil
}
