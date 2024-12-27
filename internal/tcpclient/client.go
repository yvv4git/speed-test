package tcpclient

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net"
)

type Client struct {
	logger *slog.Logger
	cfg    ClientConfig
	Conn   net.Conn
}

type ClientConfig struct {
	ServerHost string `env:"TCP_SERVER_HOST" envDefault:"127.0.0.1"`
	ServerPort uint16 `env:"TCP_SERVER_PORT" envDefault:"8080"`
	BufSize    uint16 `env:"TCP_SERVER_BUF_SIZE" envDefault:"1024"`
}

type ClientParams struct {
	Logger *slog.Logger
	Cfg    ClientConfig
	Conn   net.Conn
}

func NewClient(params ClientParams) *Client {
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

			_, err = c.Conn.Write(randomBytes)
			if err != nil {
				c.logger.Error("Failed to send random bytes", "error", err)
				return err
			}

			_, err = c.Conn.Read(buf)
			if err != nil {
				c.logger.Error("Failed to read response", "error", err)
				return err
			}
		}
	}
}

func (c *Client) Close() error {
	if c.Conn != nil {
		err := c.Conn.Close()
		if err != nil {
			return err
		}

		c.logger.Info("Connection closed")
	}

	return nil
}
