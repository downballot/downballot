package testutils

import (
	"log/slog"
	"os"
	"testing"
)

// Setup testing things.
//
// This currently initializes the logger with debugging enabled (if requested by the environment).
func Setup(t *testing.T) {
	if value := os.Getenv("DEBUG"); value == "1" || value == "true" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}
}
