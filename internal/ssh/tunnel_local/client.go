package tunnel_local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	LocalHost  string `env:"SSH_LOCAL_HOST" envDefault:"127.0.0.1"`
	LocalPort  uint16 `env:"SSH_LOCAL_PORT" envDefault:"2222"`
	ServerHost string `env:"SSH_SERVER_HOST" envDefault:"localhost"`
	ServerPort uint16 `env:"SSH_SERVER_PORT" envDefault:"22"`
	ServerUser string `env:"SSH_SERVER_USER" envDefault:"root"`
	ServerPass string `env:"SSH_SERVER_PASS" envDefault:"secret"`
	RemoteHost string `env:"SSH_REMOTE_HOST" envDefault:"127.0.0.1"`
	RemotePort uint16 `env:"SSH_REMOTE_PORT" envDefault:"1544"`
}

type Client struct {
	cfg    Config
	logger *slog.Logger
}

type Params struct {
	logger *slog.Logger
	cfg    Config
}

func NewClient(params *Params) *Client {
	return &Client{
		logger: params.logger,
		cfg:    params.cfg,
	}
}

func (c *Client) Start(ctx context.Context) error {
	sshConfig := &ssh.ClientConfig{
		User: c.cfg.ServerUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.cfg.ServerPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshAddr := fmt.Sprintf("%s:%d", c.cfg.ServerHost, c.cfg.ServerPort)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial SSH server: %w", err)
	}
	defer sshClient.Close()

	localAddr := fmt.Sprintf("%s:%d", c.cfg.LocalHost, c.cfg.LocalPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on local address %s: %w", localAddr, err)
	}

	go func() {
		<-ctx.Done()
		c.logger.Info("Closing listener")
		listener.Close()
	}()

	c.logger.Info("Local SSH tunnel established",
		slog.String("listen", localAddr),
		slog.String("forward_to", fmt.Sprintf("%s:%d", c.cfg.RemoteHost, c.cfg.RemotePort)),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Shutting down SSH tunnel client")
			return ctx.Err()
		default:
			if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second)); err != nil {
				c.logger.Warn("Failed to set deadline on listener", "error", err)
				continue
			}

			conn, errAccept := listener.Accept()
			if errAccept != nil {
				var netErr net.Error
				if errors.As(errAccept, &netErr) && netErr.Timeout() {
					continue
				}

				c.logger.Warn("Failed to accept connection", "error", errAccept)

				continue
			}

			go c.handleConnection(ctx, sshClient, conn)
		}
	}
}

func (c *Client) handleConnection(ctx context.Context, sshClient *ssh.Client, conn net.Conn) {
	defer conn.Close()

	remoteAddr := fmt.Sprintf("%s:%d", c.cfg.RemoteHost, c.cfg.RemotePort)
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		c.logger.Warn("Failed to dial remote through SSH", "error", err)
		return
	}
	defer remoteConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		if _, err = io.Copy(remoteConn, conn); err != nil && err != io.EOF {
			c.logger.Debug("Error copying to remote", "error", err)
		}
		done <- struct{}{}
	}()

	go func() {
		if _, err = io.Copy(conn, remoteConn); err != nil && err != io.EOF {
			c.logger.Debug("Error copying from remote", "error", err)
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
	case <-done:
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond): // Небольшой тайм-аут для второго io.Copy
		}
	}
}
