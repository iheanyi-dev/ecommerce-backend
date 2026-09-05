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
		// RefreshUserUseCase coordinates refresh-token rotation.
		//
		// It depends exclusively on application ports:
		//   - RefreshTokenRepository
		//   - UserRepository
		//   - RefreshTokenService
		//   - TokenService
		//
		// Infrastructure details remain outside the application layer.
		fx.Annotate(
			func(
				refreshTokenRepository ports.RefreshTokenRepository,
				userRepository ports.UserRepository,
				refreshTokenService ports.RefreshTokenService,
				tokenService ports.TokenService,
			) *use_cases.RefreshUserUseCase {
				return use_cases.NewRefreshUserUseCase(
					refreshTokenRepository,
					userRepository,
					refreshTokenService,
					tokenService,
				)
			},
			fx.As(new(ports.RefreshUserService)),
		),
		// LogoutUserUseCase coordinates revocation of the specific
		// refresh-token session being logged out.
		//
		// It depends only on application ports, keeping the use case
		// independent of HTTP and infrastructure details.
		fx.Annotate(
			func(
				refreshTokenRepository ports.RefreshTokenRepository,
				refreshTokenService ports.RefreshTokenService,
			) *use_cases.LogoutUserUseCase {
				return use_cases.NewLogoutUserUseCase(
					refreshTokenRepository,
					refreshTokenService,
				)
			},
			fx.As(new(ports.LogoutUserService)),
		),
	),
)
