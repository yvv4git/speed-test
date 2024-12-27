package quicclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"

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
	// Загружаем переменные окружения из .env файла (если есть)
	if err := godotenv.Load(); err != nil {
		a.logger.Debug("Failed to load .env file", "error", err)
	}

	// Парсим конфигурацию из переменных окружения
	var cfg ClientConfig
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	a.logger.Info("Starting QUIC client", slog.String("Host", cfg.ServerHost), slog.Int("Port", int(cfg.ServerPort)))

	// Настройка TLS (используем InsecureSkipVerify для тестирования)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}

	// Подключение к серверу
	addr := net.JoinHostPort(cfg.ServerHost, fmt.Sprintf("%d", cfg.ServerPort))
	conn, err := quic.DialAddr(ctx, addr, tlsConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Создаем клиент
	client := NewClient(ClientParams{
		Logger: a.logger,
		Cfg:    cfg,
		Conn:   conn,
	})
	defer client.Close()

	// Настройка graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// Запуск клиента (блокирующий режим с graceful shutdown)
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("client failed: %w", err)
	}

	a.logger.Info("Application stopped gracefully")
	return nil
}
