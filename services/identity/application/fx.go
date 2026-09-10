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
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				passwordHasher ports.PasswordHasher,
				logger ports.Logger,
			) *use_cases.RegisterUserUseCase {
				return use_cases.NewRegisterUserUseCase(
					userRepository,
					passwordHasher,
					logger,
				)
			},
			fx.As(new(ports.RegisterUserService)),
		),

		// AuthenticateUserUseCase coordinates the authentication workflow.
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				passwordHasher ports.PasswordHasher,
				tokenService ports.TokenService,
				logger ports.Logger,
			) *use_cases.AuthenticateUserUseCase {
				return use_cases.NewAuthenticateUserUseCase(
					userRepository,
					passwordHasher,
					tokenService,
					logger,
				)
			},
			fx.As(new(ports.AuthenticateUserService)),
		),

		// RefreshUserUseCase coordinates refresh-token rotation.
		fx.Annotate(
			func(
				refreshTokenRepository ports.RefreshTokenRepository,
				userRepository ports.UserRepository,
				refreshTokenService ports.RefreshTokenService,
				tokenService ports.TokenService,
				logger ports.Logger,
			) *use_cases.RefreshUserUseCase {
				return use_cases.NewRefreshUserUseCase(
					refreshTokenRepository,
					userRepository,
					refreshTokenService,
					tokenService,
					logger,
				)
			},
			fx.As(new(ports.RefreshUserService)),
		),

		// LogoutUserUseCase coordinates revocation of the specific
		// refresh-token session being logged out.
		fx.Annotate(
			func(
				refreshTokenRepository ports.RefreshTokenRepository,
				refreshTokenService ports.RefreshTokenService,
				logger ports.Logger,
			) *use_cases.LogoutUserUseCase {
				return use_cases.NewLogoutUserUseCase(
					refreshTokenRepository,
					refreshTokenService,
					logger,
				)
			},
			fx.As(new(ports.LogoutUserService)),
		),

		// UpdateUserProfileUseCase coordinates self-service profile updates.
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				logger ports.Logger,
			) *use_cases.UpdateUserProfileUseCase {
				return use_cases.NewUpdateUserProfileUseCase(
					userRepository,
					logger,
				)
			},
			fx.As(new(ports.UpdateUserProfileService)),
		),

		// ListUsersUseCase coordinates administrative user listing.
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				logger ports.Logger,
			) *use_cases.ListUsersUseCase {
				return use_cases.NewListUsersUseCase(
					userRepository,
					logger,
				)
			},
			fx.As(new(ports.ListUsersService)),
		),

		// GetUserUseCase coordinates administrative retrieval of a single user.
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				logger ports.Logger,
			) *use_cases.GetUserUseCase {
				return use_cases.NewGetUserUseCase(
					userRepository,
					logger,
				)
			},
			fx.As(new(ports.GetUserService)),
		),

		// UpdateUserStatusUseCase coordinates administrative account-status
		// updates.
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				logger ports.Logger,
			) *use_cases.UpdateUserStatusUseCase {
				return use_cases.NewUpdateUserStatusUseCase(
					userRepository,
				)
			},
			fx.As(new(ports.UpdateUserStatusService)),
		),

		// ChangePasswordUseCase coordinates authenticated password changes.
		fx.Annotate(
			func(
				userRepository ports.UserRepository,
				passwordHasher ports.PasswordHasher,
				logger ports.Logger,
			) *use_cases.ChangePasswordUseCase {
				return use_cases.NewChangePasswordUseCase(
					userRepository,
					passwordHasher,
					logger,
				)
			},
			fx.As(new(ports.ChangePasswordService)),
		),
	),
)
