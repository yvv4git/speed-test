package server

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

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	a.logger.Info("Loaded configuration", "host", cfg.Host, "port", cfg.Port)

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return fmt.Errorf("generate TLS config: %w", err)
	}

	quicConfig := &quic.Config{
		HandshakeIdleTimeout:  30 * time.Second, // Увеличьте таймаут рукопожатия
		MaxIdleTimeout:        60 * time.Second, // Увеличьте таймаут бездействия
		MaxIncomingStreams:    100,              // Максимальное количество входящих потоков
		MaxIncomingUniStreams: 100,              // Максимальное количество входящих однонаправленных потоков
		KeepAlivePeriod:       10 * time.Second, // Период отправки keep-alive пакетов
		EnableDatagrams:       true,             // Включить поддержку датаграмм
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	listener, err := quic.ListenAddr(addr, tlsConfig, quicConfig)
	if err != nil {
		return fmt.Errorf("start QUIC listener: %w", err)
	}

	a.logger.Info("QUIC server started", "address", addr)

	srv := NewServer(Params{
		Logger:   a.logger,
		Cfg:      cfg,
		Listener: listener,
	})

	srv.SetHandler(func(data []byte, stream quic.Stream, remoteAddr string) []byte {
		return data
	})

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		if err := srv.Start(ctx); err != nil {
			a.logger.Error("QUIC server failed", "error", err)
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
		NextProtos:   []string{"quic-echo"},
	}, nil
}
