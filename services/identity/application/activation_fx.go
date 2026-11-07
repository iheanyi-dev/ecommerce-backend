package application

import (
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
)

func provideRequestActivationUseCase(
	userRepository ports.UserRepository,
	tokenRepository ports.ActivationTokenRepository,
	emailSender ports.ActivationEmailSender,
	cfg *config.Config,
) *use_cases.RequestActivationUseCase {
	return use_cases.NewRequestActivationUseCase(
		userRepository,
		tokenRepository,
		emailSender,
		cfg.ActivationURL,
		cfg.ActivationTokenTTL,
	)
}

// provideAuthenticateUserUseCase wires activation delivery into authentication.
func provideAuthenticateUserUseCase(
	userRepository ports.UserRepository,
	passwordHasher ports.PasswordHasher,
	tokenService ports.TokenService,
	logger ports.Logger,
	metrics ports.Metrics,
	requestActivation *use_cases.RequestActivationUseCase,
) *use_cases.AuthenticateUserUseCase {
	authenticator := use_cases.NewAuthenticateUserUseCase(userRepository, passwordHasher, tokenService, logger, metrics)
	authenticator.SetRequestActivationUseCase(requestActivation)
	return authenticator
}
