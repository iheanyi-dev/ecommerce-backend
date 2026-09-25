package http

import (
	nethttp "net/http"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/proxy"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
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
	requestIDMiddleware *middleware.RequestIDMiddleware,
	requestObservabilityMiddleware *middleware.RequestObservabilityMiddleware,
	identityPropagationMiddleware *middleware.IdentityPropagationMiddleware,
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
