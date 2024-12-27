package tcpclient

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Application struct {
	logger *slog.Logger
}

func NewApplication(log *slog.Logger) *Application {
	return &Application{
		logger: log,
	}
}

func (a *Application) Start(ctx context.Context) error {
	if err := godotenv.Load(); err != nil {
		a.logger.Debug("Failed to load .env file", "error", err)
	}

	var cfg ClientConfig
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	client := NewClient(ClientParams{
		Logger: a.logger,
		Cfg:    cfg,
		Conn:   conn,
	})
	defer client.Close()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// Blocking mode, but with graceful shutdown
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("client failed: %w", err)
	}

	a.logger.Info("Application stopped gracefully")
	return nil
}