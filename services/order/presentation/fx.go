package presentation

import (
	"net/http"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/middleware"
)

// Module wires Order HTTP handlers, middleware, and routing.
var Module = fx.Module(
	"order-presentation",
	fx.Provide(
		handlers.NewOrderHandlers,
		middleware.NewServiceAuthenticationMiddleware,
		middleware.NewTracingMiddleware,
		middleware.NewRequestObservabilityMiddleware,
		fx.Annotate(
			NewRouter,
			fx.As(new(http.Handler)),
		),
	),
)
