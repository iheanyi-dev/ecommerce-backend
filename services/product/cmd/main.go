package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"

	productv1 "github.com/iheanyi-dev/ecommerce-backend/proto/product/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure"
	productgrpc "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/grpc"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
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
	address := net.JoinHostPort("", cfg.AppPort)
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", address)
			if err != nil {
				return fmt.Errorf("listen on Product HTTP %s: %w", address, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					fmt.Printf("Product HTTP server error: %v\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}

// startGRPCServer runs Product's authenticated internal API independently
// from its public HTTP server.
func startGRPCServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	service *productgrpc.Server,
) {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			productgrpc.NewServiceAuthInterceptor(
				cfg.GRPCServiceAuthName,
				cfg.GRPCServiceAuthSecret,
			),
		),
	)
	productv1.RegisterProductServiceServer(server, service)

	address := net.JoinHostPort("", cfg.GRPCPort)
	var listener net.Listener

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error
			listener, err = net.Listen("tcp", address)
			if err != nil {
				return fmt.Errorf("listen on Product gRPC %s: %w", address, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil {
					fmt.Printf("Product gRPC server error: %v\\n", err)
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
