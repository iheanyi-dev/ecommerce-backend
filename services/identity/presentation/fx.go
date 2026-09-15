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

		// Distributed tracing middleware creates an HTTP server span
		// and propagates its context to downstream handlers.
		middleware.NewTracingMiddleware,

		// Request-level observability middleware records the final
		// HTTP outcome of each request.
		middleware.NewRequestObservabilityMiddleware,

		// Service authentication middleware verifies that requests
		// reaching Identity originate from a trusted internal service.
		middleware.NewServiceAuthenticationMiddleware,

		// Request-ID middleware establishes the correlation identifier
		// used across Gateway and downstream services.
		middleware.NewRequestIDMiddleware,

		// HTTP router.
		//
		// NewRouter receives middleware dependencies explicitly;
		// dependencies are not constructed internally by the router.
		NewRouter,
	),
)
