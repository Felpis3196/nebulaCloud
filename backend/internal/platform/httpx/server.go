package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/nebulacloud/nebula/internal/platform/logger"
)

// ServerOptions configures the lifecycle wrapper around http.Server.
type ServerOptions struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Server is a graceful-shutdown-aware http.Server wrapper. It exposes Run
// which blocks until ctx is cancelled or the underlying listener fails.
type Server struct {
	srv             *http.Server
	shutdownTimeout time.Duration
}

// NewServer builds a Server with sane defaults applied for any zero-valued
// timeout. The handler is mounted as-is — middleware should already be
// composed by the caller.
func NewServer(handler http.Handler, opts ServerOptions) *Server {
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 15 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 15 * time.Second
	}
	if opts.IdleTimeout == 0 {
		opts.IdleTimeout = 60 * time.Second
	}
	if opts.ShutdownTimeout == 0 {
		opts.ShutdownTimeout = 20 * time.Second
	}
	return &Server{
		srv: &http.Server{
			Addr:              opts.Address,
			Handler:           handler,
			ReadTimeout:       opts.ReadTimeout,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      opts.WriteTimeout,
			IdleTimeout:       opts.IdleTimeout,
			BaseContext:       func(_ net.Listener) context.Context { return context.Background() },
		},
		shutdownTimeout: opts.ShutdownTimeout,
	}
}

// Run starts listening and blocks until ctx is cancelled. It then performs
// a graceful shutdown bounded by the configured timeout.
func (s *Server) Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	errCh := make(chan error, 1)

	go func() {
		log.Info("http.server.start", slog.String("address", s.srv.Addr))
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		log.Info("http.server.shutdown.begin")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http server shutdown: %w", err)
		}
		log.Info("http.server.shutdown.done")
		return nil
	case err := <-errCh:
		return err
	}
}
