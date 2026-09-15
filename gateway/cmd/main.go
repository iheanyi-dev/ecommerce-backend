package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/gateway"
	gatewayhttp "github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/http"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		gateway.Module,
		fx.Invoke(registerLifecycle),
	)

	app.Run()
}

// registerLifecycle connects the Gateway HTTP server to the Fx application
// lifecycle.
//
// Fx starts the HTTP server when the application starts and gracefully shuts
// it down when the application receives its shutdown signal.
func registerLifecycle(
	lifecycle fx.Lifecycle,
	server *gatewayhttp.Server,
) {
	lifecycle.Append(
		fx.Hook{
			OnStart: func(_ context.Context) error {
				go func() {
					if err := server.Start(); err != nil &&
						!errors.Is(err, http.ErrServerClosed) {
						os.Exit(1)
					}
				}()

				return nil
			},
			OnStop: func(ctx context.Context) error {
				shutdownCtx, cancel := context.WithTimeout(
					ctx,
					10*time.Second,
				)
				defer cancel()

				return server.Shutdown(shutdownCtx)
			},
		},
	)
}
