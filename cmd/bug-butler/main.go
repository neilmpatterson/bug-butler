package main

import (
	"log/slog"
	"os"

	"github.com/neilmpatterson/bug-butler/internal/cli"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Execute root command
	if err := cli.Execute(); err != nil {
		slog.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}
