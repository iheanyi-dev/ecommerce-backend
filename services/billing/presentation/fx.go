package presentation

import (
	grpcserver "github.com/iheanyi-dev/ecommerce-backend/services/billing/presentation/grpc"
	"go.uber.org/fx"
)

// Module provides and starts the Billing presentation layer.
var Module = fx.Module(
	"billing-presentation",
	fx.Provide(grpcserver.NewServer),
	fx.Invoke(grpcserver.RegisterGRPCServer),
)
