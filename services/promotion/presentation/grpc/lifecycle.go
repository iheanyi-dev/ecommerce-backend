package grpcserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/proto/promotion/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

func Register(server *grpc.Server, promotionServer *Server) {
	promotionv1.RegisterPromotionServiceServer(server, promotionServer)
}

func RegisterGRPCServer(
	lifecycle fx.Lifecycle,
	cfg *config.Config,
	promotionServer *Server,
) {
	address := net.JoinHostPort("", cfg.GRPCPort)

	var server *grpc.Server

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", address)
			if err != nil {
				return fmt.Errorf("listen on promotion gRPC address %s: %w", address, err)
			}

			server = grpc.NewServer(
				grpc.UnaryInterceptor(
					NewServiceAuthInterceptor(
						cfg.ServiceAuthName,
						cfg.ServiceAuthSecret,
					),
				),
			)

			Register(server, promotionServer)

			go func() {
				_ = server.Serve(listener)
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if server == nil {
				return nil
			}

			stopped := make(chan struct{})
			go func() {
				server.GracefulStop()
				close(stopped)
			}()

			select {
			case <-stopped:
				return nil
			case <-ctx.Done():
				server.Stop()
				select {
				case <-stopped:
				case <-time.After(time.Second):
				}
				return ctx.Err()
			}
		},
	})
}
