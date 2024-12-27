package quicserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"

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
	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	a.logger.Info("Loaded configuration", "host", cfg.Host, "port", cfg.Port)

	// Настройка TLS
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to generate TLS config: %w", err)
	}

	// Создание QUIC-листенера
	addr := net.JoinHostPort(cfg.Host, string(rune(cfg.Port)))
	listener, err := quic.ListenAddr(addr, tlsConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to start QUIC listener: %w", err)
	}

	a.logger.Info("QUIC server started", "address", addr)

	// Создаем сервер
	srv := NewServer(ServerParams{
		Logger:   a.logger,
		Cfg:      cfg,
		Listener: listener, // Передаем указатель на listener
	})

	// Устанавливаем обработчик (эхо-ответ)
	srv.SetHandler(func(data []byte, stream quic.Stream, remoteAddr string) []byte {
		return data
	})

	// Настройка graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Запуск QUIC-сервера в отдельной горутине
	go func() {
		if err := srv.Start(ctx); err != nil {
			a.logger.Error("QUIC server failed", "error", err)
			cancel()
		}
	}()

	// // Запуск метрик (если нужно)
	// go func() {
	// 	if err := startMetricsWebServer(cfg); err != nil {
	// 		a.logger.Error("Failed to start metrics web server", "error", err)
	// 		cancel()
	// 	}
	// }()

	// Ожидание завершения
	<-ctx.Done()

	// Остановка сервера
	srv.Stop()
	a.logger.Info("Application shutdown complete")
	return nil
}

// generateTLSConfig создает самоподписанный TLS-конфиг для QUIC
func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}, nil
}

// // startMetricsWebServer запускает веб-сервер для метрик (заглушка)
// func startMetricsWebServer(cfg ServerConfig) error {
// 	// Реализуйте запуск веб-сервера для метрик, если это необходимо
// 	return nil
// }
