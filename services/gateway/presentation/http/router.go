package http

import (
	nethttp "net/http"

	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/infrastructure/proxy"
	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/gateway/presentation/middleware"
)

// NewRouter creates the root HTTP router for the Gateway.
//
// The Gateway does not decide which downstream routes require
// authentication. Instead, identity propagation runs for every
// business-service request:
//
//   - Anonymous requests are forwarded without authenticated identity.
//   - Valid JWTs are validated and trusted identity is propagated.
//   - Invalid JWTs are rejected.
//   - Downstream services decide whether authentication is required
//     for the endpoint being accessed.
//
// Middleware is applied from the inside out:
//
//	router
//	   ↓
//	Identity Propagation
//	   ↓
//	Request-ID
//	   ↓
//	Observability
//
// Request-ID runs before observability so completed request logs always
// have the correlation identifier available.
func NewRouter(
	healthHandler *handlers.HealthHandler,
	identityProxy *proxy.IdentityProxy,
	storeProxy *proxy.StoreProxy,
	storageProxy *proxy.StorageProxy,
	recommendationProxy *proxy.RecommendationProxy,
	promotionProxy *proxy.PromotionProxy,
	requestIDMiddleware *middleware.RequestIDMiddleware,
	requestObservabilityMiddleware *middleware.RequestObservabilityMiddleware,
	identityPropagationMiddleware *middleware.IdentityPropagationMiddleware,
	productProxy *proxy.ProductProxy,
	cartProxy *proxy.CartProxy,
	orderProxy *proxy.OrderProxy,
	paymentProxy *proxy.PaymentProxy,
) nethttp.Handler {
	mux := nethttp.NewServeMux()

	// Gateway-owned health endpoint.
	mux.Handle("/health", healthHandler)

	// Identity owns its authentication boundary.
	//
	// The identity propagation middleware explicitly bypasses these
	// routes so Authorization and other request headers reach Identity
	// unchanged. Identity performs its own JWT validation.
	mux.Handle("/api/v1/users/", identityProxy)

	// Store owns the Store API namespace.
	//
	// Store is still protected by its own service-authentication boundary.
	// The Gateway supplies those credentials through StoreProxy.
	//
	// Identity propagation runs before this proxy so a validated user
	// identity is forwarded to Store through trusted internal headers.
	mux.Handle("/api/v1/stores", storeProxy)
	mux.Handle("/api/v1/stores/", storeProxy)

	// Product owns the Product API namespace. The proxy overwrites external
	// service-authentication headers with trusted Gateway-to-Product credentials.
	mux.Handle("/api/v1/products", productProxy)
	mux.Handle("/api/v1/products/", productProxy)

	// Cart owns the Cart API namespace. The proxy supplies trusted Cart
	// service credentials while identity propagation forwards validated
	// client identity when available.
	mux.Handle("/api/v1/cart", cartProxy)
	mux.Handle("/api/v1/cart/", cartProxy)

	// Recommendation owns the recommendation API namespace.
	//
	// The Gateway supplies Recommendation's trusted service credentials
	// through RecommendationProxy while identity propagation forwards
	// validated client identity when available.
	mux.Handle("/api/v1/recommendations", recommendationProxy)
	mux.Handle("/api/v1/recommendations/", recommendationProxy)

	// Promotion owns the Promotion API namespace. The proxy supplies
	// trusted service credentials while identity propagation forwards
	// validated customer identity when available.
	mux.Handle("/api/v1/promotions", promotionProxy)
	mux.Handle("/api/v1/promotions/", promotionProxy)

	// Storage exposes only its public-object HTTP endpoint through the
	// Gateway. Storage itself remains internal-only and validates the
	// trusted service credentials supplied by StorageProxy.
	mux.Handle("/api/v1/storage", storageProxy)
	mux.Handle("/api/v1/storage/", storageProxy)

	// Order owns the complete Order API namespace. The proxy supplies
	// trusted service credentials; identity propagation forwards the
	// validated customer identity when available.
	mux.Handle("/api/v1/orders", orderProxy)
	mux.Handle("/api/v1/orders/", orderProxy)

	// Only Payment endpoints explicitly classified as Gateway-facing
	// by Payment's router are exposed here. Internal payment creation,
	// verification, payout operations, and webhooks are not proxied.
	mux.Handle("GET /api/v1/payments", paymentProxy)
	mux.Handle("GET /api/v1/payments/{paymentID}", paymentProxy)
	mux.Handle("GET /api/v1/payouts/{payoutID}", paymentProxy)

	var handler nethttp.Handler = mux

	if identityPropagationMiddleware != nil {
		handler = identityPropagationMiddleware.Middleware(handler)
	}

	if requestObservabilityMiddleware != nil {
		handler = requestObservabilityMiddleware.Middleware(handler)
	}

	if requestIDMiddleware != nil {
		handler = requestIDMiddleware.Middleware(handler)
	}

	return handler
}
