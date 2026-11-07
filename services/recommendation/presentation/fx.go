package presentation

import (
	"net/http"

	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/shared/config"
)

func newServiceAuthentication(cfg *config.Config) *middleware.ServiceAuthentication {
	return middleware.NewServiceAuthentication(cfg.ServiceName, cfg.ServiceSecret)
}

var Module = fx.Module(
	"recommendation-presentation",
	fx.Provide(
		handlers.NewRecommendStoresHandler,
		handlers.NewRecommendProductsHandler,
		middleware.NewRequestID,
		middleware.NewTracing,
		middleware.NewObservability,
		newServiceAuthentication,
		NewRouter,
		NewHTTPHandler,
	),
)

func NewHTTPHandler(router *Router) http.Handler {
	return router.Handler()
}
