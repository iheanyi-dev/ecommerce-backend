package presentation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
	"go.uber.org/fx"
)

// HTTPServer owns the Promotion HTTP server lifecycle.
type HTTPServer struct {
	server *http.Server
}

// NewHTTPServer creates the Promotion HTTP server.
func NewHTTPServer(
	cfg *config.Config,
	handler http.Handler,
) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr:              net.JoinHostPort("", cfg.HTTPPort),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// RegisterHTTPServer registers the HTTP server lifecycle with Fx.
func RegisterHTTPServer(
	lifecycle fx.Lifecycle,
	server *HTTPServer,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", server.server.Addr)
			if err != nil {
				return fmt.Errorf(
					"listen on promotion HTTP address %s: %w",
					server.server.Addr,
					err,
				)
			}

			go func() {
				if err := server.server.Serve(listener); err != nil &&
					!errors.Is(err, http.ErrServerClosed) {
					// The server lifecycle is managed by Fx.
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.server.Shutdown(ctx)
		},
	})
}
