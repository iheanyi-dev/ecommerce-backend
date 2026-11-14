package presentation

import (
	grpcserver "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/grpc"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/handlers"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"promotion-presentation",
	fx.Provide(
		handlers.NewGetPromotionByIDHandler,
		handlers.NewGetPromosForUserHandler,
		handlers.NewGetPromoForUserHandler,
		grpcserver.NewServer,
		NewRouter,
		NewHTTPServer,
	),
	fx.Invoke(
		grpcserver.RegisterGRPCServer,
		RegisterHTTPServer,
	),
)
