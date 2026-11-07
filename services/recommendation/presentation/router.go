package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/middleware"
)

// Router builds the Recommendation HTTP routes and middleware chain.
type Router struct {
	recommendStores   *handlers.RecommendStoresHandler
	recommendProducts *handlers.RecommendProductsHandler
	requestID         *middleware.RequestID
	serviceAuth       *middleware.ServiceAuthentication
	tracing           *middleware.Tracing
	observability     *middleware.Observability
}

// NewRouter creates the Recommendation router.
func NewRouter(
	recommendStores *handlers.RecommendStoresHandler,
	recommendProducts *handlers.RecommendProductsHandler,
	requestID *middleware.RequestID,
	serviceAuth *middleware.ServiceAuthentication,
	tracing *middleware.Tracing,
	observability *middleware.Observability,
) *Router {
	return &Router{
		recommendStores:   recommendStores,
		recommendProducts: recommendProducts,
		requestID:         requestID,
		serviceAuth:       serviceAuth,
		tracing:           tracing,
		observability:     observability,
	}
}

// Handler returns the configured HTTP handler.
func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /api/v1/recommendations/stores", r.recommendStores)
	mux.Handle("GET /api/v1/recommendations/products", r.recommendProducts)

	var handler http.Handler = mux
	handler = r.observability.Middleware(handler)
	handler = r.tracing.Middleware(handler)
	handler = r.serviceAuth.Middleware(handler)
	handler = r.requestID.Middleware(handler)

	return handler
}
