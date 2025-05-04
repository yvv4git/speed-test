package client

import (
	"context"
	"io"
	"log/slog"
	"net"

	"github.com/gorilla/websocket"
)

type ClientConfig struct {
	LocalBindHost string `env:"WEB_CLIENT_BIND_HOST" envDefault:"127.0.0.1"`
	LocalBindPort uint16 `env:"WEB_CLIENT_BIND_PORT" envDefault:"1234"`
	WebSocketURL  string `env:"WEB_CLIENT_WS_URL" envDefault:"ws://localhost:80/tunnel"`
	BufSize       uint16 `env:"WEB_CLIENT_BUF_SIZE" envDefault:"1024"`
}

func HandleLocalConnection(ctx context.Context, conn net.Conn, cfg ClientConfig, logger *slog.Logger) {
	defer conn.Close()

	wsConn, _, err := websocket.DefaultDialer.DialContext(ctx, cfg.WebSocketURL, nil)
	if err != nil {
		logger.Error("WebSocket dial error", "error", err)
		return
	}
	defer wsConn.Close()

	logger.Info("New tunnel opened", "to", cfg.WebSocketURL, "from", conn.RemoteAddr())

	errCh := make(chan error, 2)

	go func() {
		buf := make([]byte, cfg.BufSize)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := conn.Read(buf)
				if err != nil {
					errCh <- err
					return
				}
				if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, msg, err := wsConn.ReadMessage()
				if err != nil {
					errCh <- err
					return
				}
				if _, err := conn.Write(msg); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != io.EOF {
			logger.Warn("Connection error", "error", err)
		}
	}
}
