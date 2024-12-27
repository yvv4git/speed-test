package tcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

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

	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	a.logger.Info("Loaded configuration", "host", cfg.Host, "port", cfg.Port)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	a.logger.Info("TCP server started", "address", addr)

	srv := NewServer(ServerParams{
		Logger:   a.logger,
		Cfg:      cfg,
		listener: listener,
	})

	srv.SetHandler(func(data []byte, remoteAddr string) []byte {
		return data
	})

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		if err := srv.Start(ctx); err != nil {
			a.logger.Error("Server failed", "error", err)
			cancel()
		}
	}()

	go func() {
		if err := startMetricsWebServer(cfg); err != nil {
			a.logger.Error("Failed to start metrics web server", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()

	srv.Stop()
	a.logger.Info("Application shutdown complete")
	return nil
}
