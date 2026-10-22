package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

// NewRouter registers Product HTTP endpoints and protects the service boundary.
// Public customer endpoints still require authentication from a trusted caller
// such as the API Gateway; end-user identity is required only by owner routes.
func NewRouter(
	handler *handlers.ProductHandler,
	cfg *config.Config,
) http.Handler {
	mux := http.NewServeMux()

	// Register specific paths separately from the product-ID wildcard.
	mux.HandleFunc("POST /api/v1/products", handler.Create)
	mux.HandleFunc("GET /api/v1/products", handler.ListMarketplace)
	mux.HandleFunc("GET /api/v1/products/search", handler.Search)
	mux.HandleFunc("GET /api/v1/products/me", handler.ListMine)
	mux.HandleFunc("GET /api/v1/products/store/{storeID}", handler.ListStore)

	mux.HandleFunc("GET /api/v1/products/{id}", handler.Get)
	mux.HandleFunc("PATCH /api/v1/products/{id}", handler.Update)
	mux.HandleFunc("PUT /api/v1/products/{id}/images", handler.ChangeImage)
	mux.HandleFunc("DELETE /api/v1/products/{id}", handler.Delete)

	// Reject malformed or incomplete propagated identities, while allowing
	// anonymous identities for public customer-facing endpoints.
	var router http.Handler = mux
	router = middleware.IdentityPropagationMiddleware(router)

	// Every request must originate from a trusted internal service.
	router = middleware.NewServiceAuthenticationMiddleware(cfg)(router)

	return router
}
