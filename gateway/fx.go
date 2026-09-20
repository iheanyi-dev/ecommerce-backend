package gateway

import (
	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/proxy"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/security"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/handlers"
	gatewayhttp "github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/http"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/shared/config"
	"go.uber.org/fx"
)

// Module defines the complete dependency graph for the API Gateway.
var Module = fx.Module(
	"gateway",

	fx.Provide(
		// Application configuration.
		config.Load,

		// Observability dependencies.
		observability.NewLogger,

		// HTTP handlers.
		handlers.NewHealthHandler,

		// Request-level middleware.
		middleware.NewRequestIDMiddleware,
		middleware.NewRequestObservabilityMiddleware,

		// Gateway authentication.
		//
		// The JWT validator validates access tokens issued by the
		// Identity Service. The authentication middleware will later
		// be applied only to protected business-service routes.
		newJWTValidator,
		middleware.NewAuthenticationMiddleware,
		middleware.NewIdentityPropagationMiddleware,
		// Downstream service proxies.
		newIdentityProxy,

		// HTTP server and router.
		gatewayhttp.NewRouter,
		gatewayhttp.NewServer,
	),
)

// newIdentityProxy constructs the Gateway → Identity reverse proxy.
//
// Identity is intentionally kept outside the Gateway JWT
// authentication boundary because endpoints such as login,
// registration, and token refresh must be callable before the
// client has an access token.
func newIdentityProxy(cfg *config.Config) (*proxy.IdentityProxy, error) {
	return proxy.NewIdentityProxy(
		cfg.IdentityServiceURL,
		cfg.IdentityServiceName,
		cfg.IdentityServiceSecret,
	)
}

// newJWTValidator constructs the Gateway's access-token validator.
//
// The Gateway uses the same JWT signing secret and issuer as the
// Identity Service because Identity is responsible for issuing the
// access tokens that the Gateway validates.
func newJWTValidator(cfg *config.Config) (*security.JWTValidator, error) {
	return security.NewJWTValidator(
		cfg.JWTSecret,
		cfg.JWTIssuer,
	)
}
