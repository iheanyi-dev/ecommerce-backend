package http

import (
	"context"
	"fmt"
	nethttp "net/http"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/shared/config"
)

// Server owns the Gateway HTTP server lifecycle.
//
// Routing and request handling are intentionally kept outside this type.
// The server is responsible only for HTTP server configuration and lifecycle.
type Server struct {
	httpServer *nethttp.Server
}

// NewServer creates a Gateway HTTP server using the supplied configuration
// and HTTP handler.
func NewServer(
	cfg *config.Config,
	handler nethttp.Handler,
) *Server {
	return &Server{
		httpServer: &nethttp.Server{
			Addr:              fmt.Sprintf(":%s", cfg.AppPort),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// Start starts the Gateway HTTP server.
//
// http.ErrServerClosed is returned when the server is shut down normally.
// The caller is responsible for handling that expected condition.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the Gateway HTTP server using the supplied
// context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
