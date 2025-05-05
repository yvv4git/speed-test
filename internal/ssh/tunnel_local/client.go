package tunnel_local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
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
	// TODO: add support for SSH key authentication
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
			if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
				c.logger.Warn("Failed to set deadline on listener", "error", err)
				continue
			}

			conn, err := listener.Accept()
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					continue
				}

				c.logger.Warn("Failed to accept connection", "error", err)
				continue
			}

			go c.handleConnection(ctx, sshClient, conn)
		}
	}
}

func (c *Client) handleConnection(ctx context.Context, sshClient *ssh.Client, conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Warn("Failed to close connection", "error", err)
		}
	}()

	remoteAddr := fmt.Sprintf("%s:%d", c.cfg.RemoteHost, c.cfg.RemotePort)
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		c.logger.Warn("Failed to dial remote through SSH", "error", err)
		return
	}
	defer func() {
		if err = remoteConn.Close(); err != nil {
			c.logger.Warn("Failed to close remote connection", "error", err)
		}
	}()

	copyCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer cancel()

		_, err = io.Copy(remoteConn, conn)
		if err != nil && !isClosedConnError(err) {
			c.logger.Debug("Error copying to remote", "error", err)
		}
	}()

	go func() {
		defer wg.Done()
		defer cancel()

		_, err = io.Copy(conn, remoteConn)
		if err != nil && !isClosedConnError(err) {
			c.logger.Debug("Error copying from remote", "error", err)
		}
	}()

	select {
	case <-copyCtx.Done():
		if err = conn.Close(); err != nil {
			c.logger.Warn("Failed to close connection", "error", err)
		}

		if err = remoteConn.Close(); err != nil {
			c.logger.Warn("Failed to close remote connection", "error", err)
		}
	case <-ctx.Done():
		if err = conn.Close(); err != nil {
			c.logger.Warn("Failed to close remote connection", "error", err)
		}

		if err = remoteConn.Close(); err != nil {
			c.logger.Warn("Failed to close remote connection", "error", err)
		}
	}

	wg.Wait()
}

func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}

	if err == io.EOF {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Err.Error() == "use of closed network connection"
	}

	return false
}
