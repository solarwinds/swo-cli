// Package shared provides common functionality shared across multiple commands
package shared

import (
	"fmt"
	"os"
)

// BaseOptions contains common options used across different commands
type BaseOptions struct {
	Verbose bool   // Enable verbose output
	Token   string // API token for authentication
	APIURL  string // API URL for requests
}

// VerboseLogger provides verbose logging functionality
type VerboseLogger struct {
	enabled bool
}

// NewVerboseLogger creates a new verbose logger
func NewVerboseLogger(enabled bool) *VerboseLogger {
	return &VerboseLogger{enabled: enabled}
}

// Log prints debug information when verbose mode is enabled
func (v *VerboseLogger) Log(format string, args ...interface{}) {
	if v.enabled {
		_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
	}
}

// IsEnabled returns whether verbose logging is enabled
func (v *VerboseLogger) IsEnabled() bool {
	return v.enabled
}
