package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"go.uber.org/fx"
	"google.golang.org/grpc"

	identityv1 "github.com/iheanyi-dev/ecommerce-backend/proto/identity/v1"
	identitygrpc "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/grpc"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

func main() {
	fx.New(
		shared.Module,
		infrastructure.Module,
		application.Module,
		presentation.Module,

		fx.Invoke(
			startHTTPServer,
			startGRPCServer,
		),
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

func startGRPCServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	identityServer *identitygrpc.Server,
	tracer ports.Tracer,
) {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			identitygrpc.NewTracingInterceptor(tracer),
			identitygrpc.NewServiceAuthInterceptor(
				cfg.ServiceAuthName,
				cfg.ServiceAuthSecret,
			),
		),
	)

	identityv1.RegisterIdentityServiceServer(
		server,
		identityServer,
	)

	listener, err := net.Listen(
		"tcp",
		":"+cfg.GRPCPort,
	)
	if err != nil {
		panic(err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf(
					"identity gRPC server listening on %s",
					listener.Addr().String(),
				)

				if err := server.Serve(listener); err != nil {
					log.Printf(
						"identity gRPC server stopped: %v",
						err,
					)
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			log.Println("shutting down identity gRPC server")
			server.GracefulStop()
			return nil
		},
	})
}
