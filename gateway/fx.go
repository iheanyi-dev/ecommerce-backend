package gateway

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/observability"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/infrastructure/proxy"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/handlers"
	gatewayhttp "github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/http"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/shared/config"
)

// Module provides the Gateway dependency graph.
var Module = fx.Module(
	"gateway",

	fx.Provide(
		config.Load,

		// Structured Gateway logger.
		observability.NewLogger,

		// Gateway HTTP handlers.
		handlers.NewHealthHandler,

		// Request correlation.
		middleware.NewRequestIDMiddleware,

		// Gateway request observability.
		middleware.NewRequestObservabilityMiddleware,

		// Downstream Identity proxy.
		newIdentityProxy,

		// HTTP router and server.
		gatewayhttp.NewRouter,
		gatewayhttp.NewServer,
	),
)

// newIdentityProxy creates the Identity reverse proxy from Gateway
// configuration.
//
// The proxy itself only needs the Identity service URL. Keeping the
// configuration dependency here prevents infrastructure components from
// depending on the entire Gateway configuration object.
func newIdentityProxy(
	cfg *config.Config,
) (*proxy.IdentityProxy, error) {
	return proxy.NewIdentityProxy(
		cfg.IdentityServiceURL,
		cfg.IdentityServiceName,
		cfg.IdentityServiceSecret,
	)
}
