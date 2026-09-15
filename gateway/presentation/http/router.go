package http

import (
	nethttp "net/http"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/proxy"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
)

// NewRouter creates the root HTTP router for the Gateway.
//
// Routes owned by the Gateway itself, such as health checks, are handled
// locally. Service-owned API routes are forwarded to their corresponding
// downstream service.
//
// Middleware is applied from the inside out:
//
//   router
//      ↓
//   Request-ID
//      ↓
//   Observability
//
// Request-ID runs before observability so completed request logs always have
// the correlation identifier available.
func NewRouter(
	healthHandler *handlers.HealthHandler,
	identityProxy *proxy.IdentityProxy,
	requestIDMiddleware *middleware.RequestIDMiddleware,
	requestObservabilityMiddleware *middleware.RequestObservabilityMiddleware,
) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.Handle("/health", healthHandler)
	mux.Handle("/api/v1/users/", identityProxy)

	var handler nethttp.Handler = mux

	if requestObservabilityMiddleware != nil {
		handler = requestObservabilityMiddleware.Middleware(handler)
	}

	if requestIDMiddleware != nil {
		handler = requestIDMiddleware.Middleware(handler)
	}

	return handler
}
