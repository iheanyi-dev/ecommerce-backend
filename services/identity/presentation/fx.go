package presentation

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
)

// Module provides HTTP presentation dependencies.
var Module = fx.Module(
	"presentation",

	fx.Provide(
		handlers.NewRegisterUserHandler,
		NewRouter,
	),
)
