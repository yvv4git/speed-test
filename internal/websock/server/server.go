package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/yvv4git/speed-test/internal/metrics"
)

type Config struct {
	Host          string `env:"WEB_SERVER_HOST" envDefault:"0.0.0.0"`
	Port          uint16 `env:"WEB_SERVER_PORT" envDefault:"80"`
	HostForwardTo string `env:"WEB_FORWARD_TO_HOST" envDefault:"127.0.0.1"`
	PortForwardTo uint16 `env:"WEB_FORWARD_TO_PORT" envDefault:"1544"`
	BufSize       uint16 `env:"WEB_SERVER_BUF_SIZE" envDefault:"1024"`
	MetricsAddr   string `env:"WEB_SERVER_METRICS_ADDR" envDefault:"0.0.0.0:8080"`
}

type Server struct {
	cfg    Config
	logger *slog.Logger
}

func NewServer(cfg Config, logger *slog.Logger) *Server {
	return &Server{
		cfg:    cfg,
		logger: logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/tunnel", s.handleTunnel)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	s.logger.Info("Starting WebSocket server", "address", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down WebSocket server...")
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Error("Failed to shutdown server", "error", err)
		}
	}()

	return server.ListenAndServe()
}

var upgrade = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade error", "error", err)
		return
	}
	defer ws.Close()

	targetAddr := fmt.Sprintf("%s:%d", s.cfg.HostForwardTo, s.cfg.PortForwardTo)
	tcpConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		s.logger.Error("TCP dial error", "target", targetAddr, "error", err)
		return
	}
	defer tcpConn.Close()

	remote := r.RemoteAddr
	s.logger.Info("New WebSocket connection", "remote", remote, "forward_to", targetAddr)

	// Канал WebSocket → TCP
	go func() {
		for {
			bytesReceived, data, errReadMessage := ws.ReadMessage()
			if errReadMessage != nil {
				s.logger.Warn("WebSocket read error", "error", errReadMessage)
				return
			}

			bytesSent, errWriteMessage := tcpConn.Write(data)
			if errWriteMessage != nil {
				s.logger.Warn("TCP write error", "error", errWriteMessage)
				return
			}

			metrics.AddBytesReceived(bytesReceived)
			metrics.AddBytesSent(bytesSent)
		}
	}()

	// Канал TCP → WebSocket
	buf := make([]byte, s.cfg.BufSize)
	for {
		n, errReadBuf := tcpConn.Read(buf)
		if errReadBuf != nil {
			if err != io.EOF {
				s.logger.Warn("TCP read error", "error", errReadBuf)
			}
			break
		}

		errWriteBuf := ws.WriteMessage(websocket.BinaryMessage, buf[:n])
		if errWriteBuf != nil {
			s.logger.Warn("WebSocket write error", "error", errWriteBuf)
			break
		}

		metrics.AddBytesReceived(n)
		metrics.AddBytesSent(n)
	}
}
