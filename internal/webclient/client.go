package webclient

import (
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

func HandleLocalConnection(conn net.Conn, cfg ClientConfig, logger *slog.Logger) {
	defer conn.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial(cfg.WebSocketURL, nil)
	if err != nil {
		logger.Error("WebSocket dial error", "error", err)
		return
	}
	defer wsConn.Close()

	logger.Info("New tunnel opened", "to", cfg.WebSocketURL, "from", conn.RemoteAddr())

	// TCP → WS
	go func() {
		buf := make([]byte, cfg.BufSize)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					logger.Warn("TCP read error", "error", err)
				}
				break
			}
			if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				logger.Warn("WS write error", "error", err)
				break
			}
		}
		wsConn.Close()
	}()

	// WS → TCP
	for {
		_, msg, err := wsConn.ReadMessage()
		if err != nil {
			logger.Warn("WS read error", "error", err)
			break
		}
		if _, err := conn.Write(msg); err != nil {
			logger.Warn("TCP write error", "error", err)
			break
		}
	}
}
