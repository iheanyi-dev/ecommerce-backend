package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"go.uber.org/fx"
	gogrpc "google.golang.org/grpc"

	storagev1 "github.com/iheanyi-dev/ecommerce-backend/proto/storage/v1"
	storagegrpc "github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/grpc"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/shared"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/shared/config"
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

// startHTTPServer starts the Storage HTTP server.
//
// HTTP is intentionally retained only for public-object retrieval. All
// mutating and private Storage operations are internal gRPC operations.
func startHTTPServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	router *presentation.Router,
) {
	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router.Handler(),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf(
					"storage HTTP server listening on %s",
					server.Addr,
				)

				if err := server.ListenAndServe(); err != nil &&
					err != http.ErrServerClosed {
					log.Printf(
						"storage HTTP server stopped: %v",
						err,
					)
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			log.Println("shutting down storage HTTP server")
			return server.Shutdown(ctx)
		},
	})
}

// startGRPCServer starts the internal Storage gRPC server.
//
// Every Storage gRPC operation is protected by service authentication.
// Tracing is applied at the transport boundary before requests reach the
// application layer.
func startGRPCServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	storageServer *storagegrpc.Server,
	tracer ports.Tracer,
) error {
	server := gogrpc.NewServer(
		gogrpc.ChainUnaryInterceptor(
			storagegrpc.NewTracingInterceptor(tracer),
			storagegrpc.NewServiceAuthInterceptor(
				cfg.ServiceAuthName,
				cfg.ServiceAuthSecret,
			),
		),
		gogrpc.ChainStreamInterceptor(
			storagegrpc.NewStreamTracingInterceptor(tracer),
			storagegrpc.NewServiceAuthStreamInterceptor(
				cfg.ServiceAuthName,
				cfg.ServiceAuthSecret,
			),
		),
	)

	storagev1.RegisterStorageServiceServer(
		server,
		storageServer,
	)

	listener, err := net.Listen(
		"tcp",
		":"+cfg.GRPCPort,
	)
	if err != nil {
		return err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf(
					"storage gRPC server listening on %s",
					listener.Addr().String(),
				)

				if err := server.Serve(listener); err != nil {
					log.Printf(
						"storage gRPC server stopped: %v",
						err,
					)
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			log.Println("shutting down storage gRPC server")
			server.GracefulStop()
			return nil
		},
	})

	return nil
}
