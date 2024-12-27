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
	if err := godotenv.Load(); err != nil {
		a.logger.Debug("Failed to load .env file", "error", err)
	}

	var cfg ServerConfig
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	a.logger.Info("Loaded configuration", "host", cfg.Host, "port", cfg.Port)

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to generate TLS config: %w", err)
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	listener, err := quic.ListenAddr(addr, tlsConfig, nil)
	if err != nil {
		return fmt.Errorf("failed to start QUIC listener: %w", err)
	}

	a.logger.Info("QUIC server started", "address", addr)

	srv := NewServer(ServerParams{
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

	<-ctx.Done()

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
