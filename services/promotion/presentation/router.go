package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
)

func NewRouter(
	getPromotionByIDHandler *handlers.GetPromotionByIDHandler,
	getPromosForUserHandler *handlers.GetPromosForUserHandler,
	getPromoForUserHandler *handlers.GetPromoForUserHandler,
	cfg *config.Config,
	tracer ports.Tracer,
) http.Handler {
	mux := http.NewServeMux()

	// Only service-to-service discovery requires service credentials.
	serviceAuth := middleware.NewServiceAuthenticationMiddleware(cfg)
	mux.Handle(
		"GET /api/v1/promotions/{id}",
		serviceAuth.RequireServiceAuthentication(getPromotionByIDHandler),
	)

	// End-user routes require propagated user identity instead.
	mux.Handle(
		"GET /api/v1/promotions/users/me",
		middleware.RequireAuthentication(getPromosForUserHandler),
	)
	mux.Handle(
		"GET /api/v1/promotions/best",
		middleware.RequireAuthentication(getPromoForUserHandler),
	)

	var handler http.Handler = mux
	handler = middleware.NewIdentityPropagationMiddleware().Middleware(handler)
	handler = middleware.NewRequestObservabilityMiddleware(nil, nil).Middleware(handler)

	if tracer != nil {
		handler = middleware.NewTracingMiddleware(tracer).Middleware(handler)
	}

	handler = middleware.NewRequestIDMiddleware().Middleware(handler)

	return handler
}
