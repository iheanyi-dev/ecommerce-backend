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
// authentication. Instead, the identity propagation middleware
// runs for every request:
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

	var handler nethttp.Handler = mux

	// Identity propagation is intentionally global.
	//
	// The Gateway does not maintain a list of protected routes.
	// Instead, downstream services decide whether the authenticated
	// identity is required for a particular endpoint.
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
