package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
)

func main() {
	fx.New(
		infrastructure.Module,
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
		Addr:              net.JoinHostPort("", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return fmt.Errorf("listen on Payment HTTP %s: %w", server.Addr, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					fmt.Printf("Payment HTTP server error: %v\n", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return server.Shutdown(shutdownCtx)
		},
	})
}
