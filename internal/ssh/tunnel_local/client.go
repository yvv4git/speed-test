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

// Config и Client остаются без изменений
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
		User:            c.cfg.ServerUser,
		Auth:            []ssh.AuthMethod{ssh.Password(c.cfg.ServerPass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	sshAddr := fmt.Sprintf("%s:%d", c.cfg.ServerHost, c.cfg.ServerPort)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH connection failed to %s: %w", sshAddr, err)
	}
	defer func() {
		if err = sshClient.Close(); err != nil {
			c.logger.Error("SSH client close error", "error", err)
		}
	}()

	localAddr := fmt.Sprintf("%s:%d", c.cfg.LocalHost, c.cfg.LocalPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", localAddr, err)
	}

	go func() {
		<-ctx.Done()
		if err = listener.Close(); err != nil {
			c.logger.Error("Listener close error", "error", err)
		}
	}()

	c.logger.Info("SSH tunnel started",
		"local", localAddr,
		"remote", fmt.Sprintf("%s:%d", c.cfg.RemoteHost, c.cfg.RemotePort))

	for {
		conn, errAccept := listener.Accept()
		if errAccept != nil {
			if errors.Is(errAccept, net.ErrClosed) {
				return nil
			}

			c.logger.Error("Accept connection error", "error", errAccept)
			continue
		}

		go func(conn net.Conn) {
			if err = c.handleConnection(ctx, sshClient, conn); err != nil {
				c.logger.Error("Connection handler error", "error", err)
			}
		}(conn)
	}
}

func (c *Client) handleConnection(ctx context.Context, sshClient *ssh.Client, localConn net.Conn) error {
	defer func() {
		if err := localConn.Close(); err != nil && !isNetClosedError(err) {
			c.logger.Warn("Local connection close warning", "error", err)
		}
	}()

	remoteAddr := fmt.Sprintf("%s:%d", c.cfg.RemoteHost, c.cfg.RemotePort)
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		return fmt.Errorf("remote dial to %s: %w", remoteAddr, err)
	}
	defer func() {
		if err = remoteConn.Close(); err != nil && !isNetClosedError(err) {
			c.logger.Warn("Remote connection close warning", "error", err)
		}
	}()

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err = io.Copy(remoteConn, localConn)
		if err != nil && !isNetClosedError(err) {
			errCh <- fmt.Errorf("local->remote copy error: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		_, err = io.Copy(localConn, remoteConn)
		if err != nil && !isNetClosedError(err) {
			errCh <- fmt.Errorf("remote->local copy error: %w", err)
		}
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case errCopy, ok := <-errCh:
		if ok && errCopy != nil {
			return errCopy
		}

		return nil
	}
}

func isNetClosedError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF)
}
