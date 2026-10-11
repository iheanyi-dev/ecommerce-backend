package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		infrastructure.Module,
		application.Module,
		presentation.Module,
		fx.Invoke(startHTTPServer),
	).Run()
}

func startHTTPServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	handler http.Handler,
) {
	server := &http.Server{
		Addr:              net.JoinHostPort("", cfg.AppPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return fmt.Errorf("listen on %s: %w", server.Addr, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					// Serve errors are returned during shutdown; Fx handles lifecycle.
					fmt.Printf("Store HTTP server error: %v\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}
