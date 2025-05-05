package main

import (
	"context"
	"log/slog"
	"os"

	tunnel "github.com/yvv4git/speed-test/internal/ssh/tunnel_local"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	app := tunnel.NewApplication(logger)

	if err := app.Start(context.Background()); err != nil {
		logger.Error("Application exited with error", "error", err)
		os.Exit(1)
	}
}
