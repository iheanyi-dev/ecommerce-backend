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
		// RegisterUserUseCase coordinates the user registration workflow.
		//
		// Its dependencies are application ports, keeping the application
		// layer independent of concrete infrastructure implementations.
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

		// AuthenticateUserUseCase coordinates the authentication workflow.
		//
		// Authentication requires:
		//   - UserRepository to find the user.
		//   - PasswordHasher to verify the password.
		//   - TokenService to generate the access token.
		//
		// All three are application ports, so this use case remains
		// independent of PostgreSQL, bcrypt, JWT, and other infrastructure.
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				passwordHasher ports.PasswordHasher,
				tokenService ports.TokenService,
			) *use_cases.AuthenticateUserUseCase {
				return use_cases.NewAuthenticateUserUseCase(
					userRepository,
					passwordHasher,
					tokenService,
				)
			},
			fx.As(new(ports.AuthenticateUserService)),
		),
	),
)
