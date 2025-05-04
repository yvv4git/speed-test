package webclient

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
		a.logger.Debug("load .env file", "error", err)
	}

	var cfg ClientConfig
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.LocalBindHost, cfg.LocalBindPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind TCP: %w", err)
	}
	defer listener.Close()

	a.logger.Info("Listening for local TCP connections",
		slog.String("addr", addr),
		slog.String("forward_to_ws", cfg.WebSocketURL),
	)

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Shutting down WebSocket client")
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				a.logger.Error("Failed to accept connection", "error", err)
				continue
			}
			go HandleLocalConnection(conn, cfg, a.logger)
		}
	}
}
