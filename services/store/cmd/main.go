package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	storev1 "github.com/iheanyi-dev/ecommerce-backend/proto/store/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure"
	storegrpc "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/grpc"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

func main() {
	fx.New(
		infrastructure.Module,
		application.Module,
		presentation.Module,
		fx.Invoke(startHTTPServer),
		fx.Invoke(startGRPCServer),
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

// startGRPCServer runs Store's authenticated internal gRPC API independently
// from its public HTTP server.
func startGRPCServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	service *storegrpc.Server,
) {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			storegrpc.NewServiceAuthInterceptor(
				cfg.ServiceAuthName,
				cfg.ServiceAuthSecret,
			),
		),
	)
	storev1.RegisterStoreServiceServer(server, service)

	address := net.JoinHostPort("", cfg.GRPCPort)
	var listener net.Listener

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error
			listener, err = net.Listen("tcp", address)
			if err != nil {
				return fmt.Errorf("listen on Store gRPC %s: %w", address, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil {
					fmt.Printf("Store gRPC server error: %v\\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.GracefulStop()
			return nil
		},
	})
}
