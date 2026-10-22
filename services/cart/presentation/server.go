package presentation

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/shared/config"
	"go.uber.org/fx"
)

// RegisterHTTPServerLifecycle manages the Cart HTTP server through Fx.
func RegisterHTTPServerLifecycle(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
	handler http.Handler,
) {
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	var listener net.Listener

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", cfg.HTTPAddr)
			if err != nil {
				return err
			}
			listener = ln

			go func() {
				err := server.Serve(ln)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					// The application logger can report unexpected Serve errors.
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if listener == nil {
				return nil
			}
			return server.Shutdown(ctx)
		},
	})
}
