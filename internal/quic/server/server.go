package server

import (
	"context"
	"log/slog"
	"sync"

	"github.com/quic-go/quic-go"
)

type HandlerFunc func(data []byte, stream quic.Stream, remoteAddr string) []byte

type Server struct {
	cfg      Config
	listener *quic.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *slog.Logger
	handler  HandlerFunc
}

type Config struct {
	Host        string `env:"QUIC_SERVER_HOST" envDefault:"0.0.0.0"`
	Port        uint16 `env:"QUIC_SERVER_PORT" envDefault:"1543"`
	BufSize     uint16 `env:"QUIC_SERVER_BUF_SIZE" envDefault:"1024"`
	MetricsAddr string `env:"QUIC_SERVER_METRICS_ADDR" envDefault:"0.0.0.0:8080"`
}

type Params struct {
	Cfg      Config
	Logger   *slog.Logger
	Listener *quic.Listener
}

func NewServer(params Params) *Server {
	return &Server{
		cfg:      params.Cfg,
		logger:   params.Logger,
		listener: params.Listener,
	}
}

func (s *Server) SetHandler(handler HandlerFunc) {
	s.handler = handler
}

func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	s.logger.Info("QUIC server started", "address", s.listener.Addr())

	return s.acceptConnections() // Blocking mode
}

func (s *Server) acceptConnections() error {
	for {
		session, err := s.listener.Accept(s.ctx)
		if err != nil {
			select {
			case <-s.ctx.Done():
				s.logger.Info("QUIC server stopped accepting connections")
				return nil
			default:
				s.logger.Error("Failed to accept QUIC connection", "error", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleSession(session)
	}
}

func (s *Server) handleSession(session quic.Connection) {
	defer s.wg.Done()

	remoteAddr := session.RemoteAddr().String()
	s.logger.Info("New QUIC session", "remote_addr", remoteAddr)

	for {
		// Accepting a new thread within the session
		stream, err := session.AcceptStream(s.ctx)
		if err != nil {
			s.logger.Error("Failed to accept QUIC stream", "error", err)
			return
		}

		s.wg.Add(1)
		go s.handleStream(stream, remoteAddr)
	}
}

func (s *Server) handleStream(stream quic.Stream, remoteAddr string) {
	defer s.wg.Done()
	defer stream.Close()

	buf := make([]byte, s.cfg.BufSize)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Stream handling stopped due to server shutdown")
			return
		default:
			n, err := stream.Read(buf)
			if err != nil {
				s.logger.Error("Error reading from QUIC stream", "error", err)
				return
			}

			bytesReceived.Add(float64(n)) // Increment bytes received counter

			if s.handler != nil {
				response := s.handler(buf[:n], stream, remoteAddr)

				if _, err = stream.Write(response); err != nil {
					s.logger.Error("Failed to send response to QUIC stream", "error", err)
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
		if err := s.listener.Close(); err != nil {
			s.logger.Error("Failed to close QUIC listener", "error", err)
		}
	}

	s.wg.Wait()
	s.logger.Info("QUIC server stopped")
}
