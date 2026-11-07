package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/shared/config"
)

func main() {
	fx.New(
		fx.Provide(config.Load),
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
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return fmt.Errorf("listen on Recommendation HTTP %s: %w", server.Addr, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					fmt.Printf("Recommendation HTTP server error: %v\n", err)
				}
			}()

			fmt.Printf("Recommendation HTTP server listening on %s\n", server.Addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}
