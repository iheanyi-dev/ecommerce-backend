package grpc

import (
	"context"
	"fmt"
	"net"

	billingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

// RegisterGRPCServer registers the Billing gRPC server with the application lifecycle.
func RegisterGRPCServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	server *Server,
) {
	var grpcServer *grpc.Server

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", net.JoinHostPort("", cfg.GRPCPort))
			if err != nil {
				return fmt.Errorf("listen for billing gRPC: %w", err)
			}

			grpcServer = grpc.NewServer(
				grpc.UnaryInterceptor(
					NewServiceAuthInterceptor(cfg.ServiceAuthName, cfg.ServiceAuthSecret),
				),
			)

			billingv1.RegisterBillingServiceServer(grpcServer, server)

			go func() {
				_ = grpcServer.Serve(listener)
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if grpcServer == nil {
				return nil
			}

			stopped := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				close(stopped)
			}()

			select {
			case <-stopped:
				return nil
			case <-ctx.Done():
				grpcServer.Stop()
				return ctx.Err()
			}
		},
	})

}
