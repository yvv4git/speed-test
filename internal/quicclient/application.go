package quicclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	"github.com/quic-go/quic-go"
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

	a.logger.Info("Starting QUIC client", slog.String("Host", cfg.ServerHost), slog.Int("Port", int(cfg.ServerPort)))

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo"},
	}

	quicConfig := &quic.Config{
		HandshakeIdleTimeout:  30 * time.Second, // Увеличьте таймаут рукопожатия
		MaxIdleTimeout:        60 * time.Second, // Увеличьте таймаут бездействия
		MaxIncomingStreams:    100,              // Максимальное количество входящих потоков
		MaxIncomingUniStreams: 100,              // Максимальное количество входящих однонаправленных потоков
		KeepAlivePeriod:       10 * time.Second, // Период отправки keep-alive пакетов
		EnableDatagrams:       true,             // Включить поддержку датаграмм
	}

	addr := net.JoinHostPort(cfg.ServerHost, fmt.Sprintf("%d", cfg.ServerPort))
	conn, err := quic.DialAddr(ctx, addr, tlsConfig, quicConfig)
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}

	client := NewClient(ClientParams{
		Logger: a.logger,
		Cfg:    cfg,
		Conn:   conn,
	})
	defer client.Close()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("client failed: %w", err)
	}

	a.logger.Info("Application stopped gracefully")
	return nil
}
