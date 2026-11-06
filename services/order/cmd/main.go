package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"
	"google.golang.org/grpc"

	orderv1 "github.com/iheanyi-dev/ecommerce-backend/proto/order/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation"
	ordergrpc "github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/grpc"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/shared/config"
)

func main() {
	fx.New(
		fx.Provide(ordergrpc.NewServer),
		fx.Provide(
			func(uc *use_cases.ListOrders) ordergrpc.ListStoreOrdersUseCase { return uc },
			func(uc *use_cases.GetOrder) ordergrpc.GetStoreOrderUseCase { return uc },
			func(uc *use_cases.AddShipping) ordergrpc.AddOrderShippingUseCase { return uc },
			func(uc *use_cases.MarkSent) ordergrpc.MarkOrderSentUseCase { return uc },
			func(uc *use_cases.CancelOrder) ordergrpc.CancelStoreOrderUseCase { return uc },
			func(uc *use_cases.InitiatePayout) ordergrpc.InitiateOrderPayoutUseCase { return uc },
		),
		infrastructure.Module,
		application.Module,
		presentation.Module,
		fx.Invoke(startHTTPServer),
		fx.Invoke(startGRPCServer),
	).Run()
}

func startHTTPServer(lc fx.Lifecycle, cfg *config.Config, handler http.Handler) {
	server := &http.Server{
		Addr:              net.JoinHostPort("", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return fmt.Errorf("listen on Order HTTP %s: %w", server.Addr, err)
			}
			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					fmt.Printf("Order HTTP server error: %v\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}

// startGRPCServer runs Order's authenticated internal gRPC API.
func startGRPCServer(lc fx.Lifecycle, cfg *config.Config, service *ordergrpc.Server) {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			ordergrpc.NewServiceAuthInterceptor(
				cfg.StoreServiceAuthName,
				cfg.StoreServiceAuthSecret,
			),
		),
	)
	orderv1.RegisterOrderServiceServer(server, service)

	var listener net.Listener
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error
			listener, err = net.Listen("tcp", cfg.GRPCAddr)
			if err != nil {
				return fmt.Errorf("listen on Order gRPC %s: %w", cfg.GRPCAddr, err)
			}
			go func() {
				if err := server.Serve(listener); err != nil {
					fmt.Printf("Order gRPC server error: %v\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
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
				<-stopped
				return ctx.Err()
			}
		},
	})
}
