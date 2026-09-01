package application

import (
	"go.uber.org/fx"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
)

// Module provides application-layer dependencies.
//
// The application layer coordinates domain behavior through use cases and
// depends only on application ports rather than infrastructure details.
var Module = fx.Module(
	"application",

	fx.Provide(
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				passwordHasher ports.PasswordHasher,
			) *use_cases.RegisterUserUseCase {
				return use_cases.NewRegisterUserUseCase(
					userRepository,
					passwordHasher,
				)
			},
			fx.As(new(ports.RegisterUserService)),
		),
	),
)