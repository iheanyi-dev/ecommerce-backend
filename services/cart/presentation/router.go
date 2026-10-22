package presentation

import (
	"net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/presentation/middleware"
)

// NewRouter builds the Cart HTTP routing tree.
func NewRouter(
	getMyCartHandler http.Handler,
	addItemToCartHandler http.Handler,
	setCartItemQuantityHandler http.Handler,
	removeCartItemHandler http.Handler,
	clearCartHandler http.Handler,
	checkoutCartHandler http.Handler,
	serviceAuthenticationMiddleware *middleware.ServiceAuthenticationMiddleware,
	identityPropagationMiddleware *middleware.IdentityPropagationMiddleware,
	requestIDMiddleware *middleware.RequestIDMiddleware,
	tracingMiddleware *middleware.TracingMiddleware,
) http.Handler {
	mux := http.NewServeMux()

	// Every route requires both trusted service authentication and a customer
	// identity. The identity is never accepted from the request body.
	mux.Handle("GET /api/v1/cart", getMyCartHandler)
	mux.Handle("POST /api/v1/cart/items", addItemToCartHandler)
	mux.Handle("PATCH /api/v1/cart/items/{itemID}", setCartItemQuantityHandler)
	mux.Handle("DELETE /api/v1/cart/items/{itemID}", removeCartItemHandler)
	mux.Handle("DELETE /api/v1/cart", clearCartHandler)
	mux.Handle("POST /api/v1/cart/checkout", checkoutCartHandler)

	var handler http.Handler = mux
	handler = identityPropagationMiddleware.Middleware(handler)
	handler = serviceAuthenticationMiddleware.RequireServiceAuthentication(handler)
	handler = requestIDMiddleware.Middleware(handler)
	handler = tracingMiddleware.Middleware(handler)

	handler = tracingMiddleware.Middleware(handler)
	return handler
}
