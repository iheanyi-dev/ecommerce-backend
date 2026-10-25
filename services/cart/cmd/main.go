package main

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/fx"
	"google.golang.org/grpc"

	cartv1 "github.com/iheanyi-dev/ecommerce-backend/proto/cart/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure"
	cartgrpc "github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/grpc"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/shared/config"
)

func main() {
	fx.New(
		infrastructure.Module,
		presentation.Module,
		fx.Invoke(startGRPCServer),
	).Run()
}

// startGRPCServer runs Cart's authenticated internal API independently from
// its existing HTTP API.
func startGRPCServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	service *cartgrpc.Server,
) {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			cartgrpc.NewServiceAuthInterceptor(
				cfg.OrderServiceAuthName,
				cfg.OrderServiceAuthSecret,
			),
		),
	)
	cartv1.RegisterCartServiceServer(server, service)

	var listener net.Listener

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error
			listener, err = net.Listen("tcp", cfg.GRPCAddr)
			if err != nil {
				return fmt.Errorf("listen on Cart gRPC %s: %w", cfg.GRPCAddr, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil {
					fmt.Printf("Cart gRPC server error: %v\\n", err)
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			server.GracefulStop()
			return nil
		},
	})
}
