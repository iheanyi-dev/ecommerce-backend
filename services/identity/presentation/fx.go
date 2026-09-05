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

		// Logout endpoint handler.
		handlers.NewLogoutUserHandler,

		// Authentication middleware validates Bearer access tokens
		// before protected handlers are executed.
		middleware.NewAuthenticationMiddleware,

		// HTTP router.
		NewRouter,
	),
)
