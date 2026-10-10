// Package server wires the handlers into an http.Server.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/i-rme/gop-server/src/handler"
	"github.com/i-rme/gop-server/src/repository/filesystem"
	"github.com/i-rme/gop-server/src/server/config"
	"github.com/i-rme/gop-server/src/server/logs"
)

// shutdownTimeout is how long requests in flight get to finish on shutdown
const shutdownTimeout = 10 * time.Second

// Server is a configured GOP server, ready to Run
type Server struct {
	http *http.Server
	log  *slog.Logger
}

// New prepares the server's directories and routes
func New(c *config.Config, log *slog.Logger) (*Server, error) {
	if err := filesystem.ResetDir(c.CacheRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.UploadsRoot, 0o755); err != nil {
		return nil, err
	}

	h, err := handler.New(c, log, &logs.Stats{})
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("/server-status/", h.ServerStatus)
	mux.HandleFunc("/gop-info/", h.GopInfo)

	return &Server{
		log: log,
		http: &http.Server{
			Addr:           net.JoinHostPort(c.Address, strconv.Itoa(c.Port)),
			Handler:        timed(mux, log),
			ReadTimeout:    c.ReadTimeout.Std(),
			WriteTimeout:   c.WriteTimeout.Std(),
			IdleTimeout:    c.IdleTimeout.Std(),
			MaxHeaderBytes: c.MaxHeaderBytes,
			ErrorLog:       slog.NewLogLogger(log.Handler(), slog.LevelError),
		},
	}, nil
}

// Run serves until ctx is done, then lets requests in flight finish
func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return err
	}
	s.log.Info("server started", "url", "http://"+s.http.Addr)

	served := make(chan error, 1)
	go func() { served <- s.http.Serve(listener) }()

	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	s.log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// timed logs how long each request took
func timed(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Debug("served", "remote", r.RemoteAddr, "path", r.URL.Path, "duration", time.Since(start))
	})
}
