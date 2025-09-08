// Package shared provides common functionality shared across multiple commands
package shared

import (
	"log/slog"
	"os"
)

// BaseOptions contains common options used across different commands
type BaseOptions struct {
	Verbose bool   // Enable verbose output
	Token   string // API token for authentication
	APIURL  string // API URL for requests
}

// SetupLogger configures the global slog logger based on verbose flag
func SetupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
}
