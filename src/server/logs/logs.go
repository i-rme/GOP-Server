// Package logs sets up the server log and counts recent requests for the status page.
package logs

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// New returns a logger writing to stdout and to a new file in dir named after the
// current time. level is the configured LoggingLevel: debug, basic or error.
func New(dir, level string) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}
	name := filepath.Join(dir, time.Now().Format("2006.01.02_15.04.05")+".log")
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("creating log file: %w", err)
	}

	handler := slog.NewTextHandler(io.MultiWriter(os.Stdout, file), &slog.HandlerOptions{Level: Level(level)})
	return slog.New(handler), file, nil
}

// Level maps a LoggingLevel to the lowest slog level it shows
func Level(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "basic":
		return slog.LevelInfo
	default:
		return slog.LevelError
	}
}
