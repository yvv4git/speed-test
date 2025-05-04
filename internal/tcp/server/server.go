package server

import (
	"context"
	"log/slog"
	"net"
	"sync"
)

type HandlerFunc func(data []byte, remoteAddr string) []byte

type Server struct {
	cfg      ServerConfig
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *slog.Logger
	handler  HandlerFunc
}

type ServerConfig struct {
	Host        string `env:"TCP_SERVER_HOST" envDefault:"0.0.0.0"`
	Port        uint16 `env:"TCP_SERVER_PORT" envDefault:"1543"`
	BufSize     uint16 `env:"TCP_SERVER_BUF_SIZE" envDefault:"1024"`
	MetricsAddr string `env:"TCP_SERVER_METRICS_ADDR" envDefault:"0.0.0.0:8080"`
}

type ServerParams struct {
	Cfg      ServerConfig
	Logger   *slog.Logger
	listener net.Listener
}

func NewServer(params ServerParams) *Server {
	return &Server{
		cfg:      params.Cfg,
		logger:   params.Logger,
		listener: params.listener,
	}
}

func (s *Server) SetHandler(handler HandlerFunc) {
	s.handler = handler
}

func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	return s.acceptConnections() // Blocking mode
}

func (s *Server) acceptConnections() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				s.logger.Info("TCP server stopped accepting connections")
				return nil
			default:
				s.logger.Error("Failed to accept connection", "error", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	s.logger.Info("New connection", "remote_addr", remoteAddr)

	buf := make([]byte, s.cfg.BufSize)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Connection handling stopped due to server shutdown")
			return
		default:
			n, err := conn.Read(buf)
			if err != nil {
				s.logger.Error("Error reading from connection", "error", err)
				return
			}

			bytesReceived.Add(float64(n)) // Increment bytes received counter

			if s.handler != nil {
				response := s.handler(buf[:n], remoteAddr)

				if n, err = conn.Write(response); err != nil {
					s.logger.Error("Failed to send response to client", "error", err)
					return
				}

				bytesSent.Add(float64(n)) // Increment bytes sent counter
			}
		}
	}
}

func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.wg.Wait()
	s.logger.Info("TCP server stopped")
}
