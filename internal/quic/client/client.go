package client

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"

	"github.com/quic-go/quic-go"
)

type Client struct {
	logger *slog.Logger
	cfg    Config
	Conn   quic.Connection // Используем quic.Connection вместо net.Conn
}

type Config struct {
	ServerHost string `env:"QUIC_CLIENT_SERVER_HOST" envDefault:"127.0.0.1"`
	ServerPort uint16 `env:"QUIC_CLIENT_SERVER_PORT" envDefault:"1543"`
	BufSize    uint16 `env:"QUIC_CLIENT_BUF_SIZE" envDefault:"1024"`
}

type Params struct {
	Logger *slog.Logger
	Cfg    Config
	Conn   quic.Connection
}

func NewClient(params Params) *Client {
	return &Client{
		logger: params.Logger,
		cfg:    params.Cfg,
		Conn:   params.Conn,
	}
}

func (c *Client) Start(ctx context.Context) error {
	if c.Conn == nil {
		return errors.New("connection is not established")
	}

	stream, err := c.Conn.OpenStreamSync(ctx)
	if err != nil {
		c.logger.Error("Failed to open stream", "error", err)
		return err
	}
	defer stream.Close()

	buf := make([]byte, c.cfg.BufSize)
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Client stopping due to context cancellation")
			return nil

		default:
			randomBytes := make([]byte, c.cfg.BufSize)
			_, err := rand.Read(randomBytes)
			if err != nil {
				c.logger.Error("Failed to generate random bytes", "error", err)
				return err
			}

			_, err = stream.Write(randomBytes)
			if err != nil {
				c.logger.Error("Failed to send random bytes", "error", err)
				return err
			}

			_, err = stream.Read(buf)
			if err != nil {
				c.logger.Error("Failed to read response", "error", err)
				return err
			}
		}
	}
}

func (c *Client) Close() error {
	if c.Conn != nil {
		err := c.Conn.CloseWithError(0, "client closing")
		if err != nil {
			return err
		}

		c.logger.Info("Connection closed")
	}

	return nil
}
