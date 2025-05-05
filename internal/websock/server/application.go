package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Application struct {
	logger *slog.Logger
}

func NewApplication(logger *slog.Logger) *Application {
	return &Application{
		logger: logger,
	}
}

func (a *Application) Start(ctx context.Context) error {
	// Загружаем переменные окружения из .env
	if err := godotenv.Load(); err != nil {
		a.logger.Debug("No .env file found", "error", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	a.logger.Info("Configuration loaded",
		"host", cfg.Host,
		"port", cfg.Port,
		"forward_to", fmt.Sprintf("%s:%d", cfg.HostForwardTo, cfg.PortForwardTo),
	)

	srv := NewServer(cfg, a.logger)

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		if err := srv.Start(ctx); err != nil {
			a.logger.Error("WebSocket server error", "error", err)
			cancel()
		}
	}()

	go func() {
		if err := startMetricsWebServer(cfg); err != nil {
			a.logger.Error("Failed to start metrics server", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	a.logger.Info("Application shutdown complete")
	return nil
}
