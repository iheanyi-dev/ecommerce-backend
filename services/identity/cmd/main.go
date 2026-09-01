package main

import (
	"context"
	"log"
	"net/http"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared"
)

func main() {
	fx.New(
		shared.Module,
		infrastructure.Module,
		application.Module,
		presentation.Module,

		fx.Invoke(startHTTPServer),
	).Run()
}

// startHTTPServer starts the Identity service HTTP server.
//
// The main package only assembles the application. It does not know about
// users, repositories, password hashing, SQLC, or PostgreSQL details.
func startHTTPServer(
	lc fx.Lifecycle,
	router http.Handler,
) {
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf(
					"identity service listening on %s",
					server.Addr,
				)

				if err := server.ListenAndServe(); err != nil &&
					err != http.ErrServerClosed {
					log.Printf(
						"identity HTTP server stopped: %v",
						err,
					)
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			log.Println("shutting down identity HTTP server")

			return server.Shutdown(ctx)
		},
	})
}
