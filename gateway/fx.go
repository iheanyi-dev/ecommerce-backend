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
		config.Load,

		observability.NewLogger,

		handlers.NewHealthHandler,

		middleware.NewRequestIDMiddleware,
		middleware.NewRequestObservabilityMiddleware,

		newJWTValidator,
		middleware.NewAuthenticationMiddleware,
		middleware.NewIdentityPropagationMiddleware,

		newIdentityProxy,
		newStoreProxy,

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

// newStoreProxy constructs the Gateway → Store reverse proxy.
//
// Store is a business service. The Gateway therefore runs its normal
// identity-propagation middleware before this proxy, allowing anonymous
// requests through while propagating a validated user identity when a
// valid access token is present.
//
// Store itself decides whether the requested endpoint requires a user.
func newStoreProxy(cfg *config.Config) (*proxy.StoreProxy, error) {
	return proxy.NewStoreProxy(
		cfg.StoreServiceURL,
		cfg.StoreServiceName,
		cfg.StoreServiceSecret,
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
