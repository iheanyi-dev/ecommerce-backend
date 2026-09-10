package presentation

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

// Module provides HTTP presentation-layer dependencies.
//
// The presentation layer contains HTTP handlers, routing, and middleware.
// Business logic remains in the application layer.
var Module = fx.Module(
	"presentation",

	fx.Provide(
		// Registration endpoint handler.
		handlers.NewRegisterUserHandler,

		// Login endpoint handler.
		handlers.NewLoginUserHandler,

		// Refresh-token endpoint handler.
		handlers.NewRefreshUserHandler,

		// Authenticated-user endpoint handler.
		handlers.NewMeHandler,

		// Authenticated-user profile update endpoint handler.
		handlers.NewUpdateUserProfileHandler,

		// Logout endpoint handler.
		handlers.NewLogoutUserHandler,

		// Administrative list-users endpoint handler.
		handlers.NewListUsersHandler,

		// Administrative get-user endpoint handler.
		handlers.NewGetUserHandler,

		// Administrative update-user-status endpoint handler.
		handlers.NewUpdateUserStatusHandler,

		// Authentication middleware validates Bearer access tokens
		// before protected handlers are executed.
		middleware.NewAuthenticationMiddleware,

		// Request-level observability middleware records the final
		// HTTP outcome of each request.
		//
		// Its logger dependency is supplied explicitly by Fx through
		// the application's Logger provider.
		middleware.NewRequestObservabilityMiddleware,

		// HTTP router.
		//
		// NewRouter receives the logger and request observability
		// middleware explicitly; neither dependency is constructed
		// internally by the router.
		NewRouter,
	),
)
