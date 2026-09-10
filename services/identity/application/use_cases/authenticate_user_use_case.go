package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// AuthenticateUserUseCase coordinates the complete authentication workflow.
//
// The use case is responsible for:
//   - validating the supplied email
//   - locating the user
//   - verifying the supplied password
//   - checking the account status
//   - generating an access token
//   - recording structured authentication events
//   - returning the authenticated user's identity
type AuthenticateUserUseCase struct {
	userRepository ports.UserRepository
	passwordHasher ports.PasswordHasher
	tokenService   ports.TokenService
	logger         ports.Logger
}

// NewAuthenticateUserUseCase creates the authentication use case.
//
// Dependencies are supplied through application ports, keeping the use
// case independent from PostgreSQL, bcrypt, JWT, HTTP, infrastructure,
// and concrete logging libraries.
func NewAuthenticateUserUseCase(
	userRepository ports.UserRepository,
	passwordHasher ports.PasswordHasher,
	tokenService ports.TokenService,
	logger ports.Logger,
) *AuthenticateUserUseCase {
	return &AuthenticateUserUseCase{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		tokenService:   tokenService,
		logger:         logger,
	}
}

// Authenticate verifies credentials, validates the account status,
// generates an access token, and records the authentication outcome.
func (u *AuthenticateUserUseCase) Authenticate(
	ctx context.Context,
	command dto.LoginUserCommand,
) (dto.LoginUserResult, error) {
	// Convert the supplied email into the domain Email value object.
	//
	// Invalid email input is treated as an authentication failure rather
	// than exposing validation details to an unauthenticated caller.
	email, err := user.NewEmail(command.Email)
	if err != nil {
		return dto.LoginUserResult{}, application_errors.ErrInvalidCredentials
	}

	// Retrieve the persisted user.
	//
	// A missing user intentionally produces the same generic error used
	// when the password is incorrect.
	authenticatedUser, err := u.userRepository.FindByEmail(ctx, email)
	if err != nil {
		u.logAuthenticationEvent(ctx, ports.LogEvent{
			Event:           "auth.login.user_not_found",
			Operation:       "login",
			FailureCategory: "user_not_found",
		})

		return dto.LoginUserResult{}, application_errors.ErrInvalidCredentials
	}

	if authenticatedUser == nil {
		u.logAuthenticationEvent(ctx, ports.LogEvent{
			Event:           "auth.login.user_not_found",
			Operation:       "login",
			FailureCategory: "user_not_found",
		})

		return dto.LoginUserResult{}, application_errors.ErrInvalidCredentials
	}

	// Verify the supplied plaintext password against the stored hash.
	if err := u.passwordHasher.Verify(
		ctx,
		command.Password,
		authenticatedUser.PasswordHash().String(),
	); err != nil {
		u.logAuthenticationEvent(ctx, ports.LogEvent{
			Event:           "auth.login.invalid_credentials",
			Operation:       "login",
			FailureCategory: "invalid_credentials",
		})

		return dto.LoginUserResult{}, application_errors.ErrInvalidCredentials
	}

	// Only active accounts may authenticate.
	if authenticatedUser.Status() != user.StatusActive {
		u.logAuthenticationEvent(ctx, ports.LogEvent{
			Event:           "auth.login.user_inactive",
			Operation:       "login",
			UserID:          authenticatedUser.ID().String(),
			Role:            authenticatedUser.Role().String(),
			FailureCategory: "user_inactive",
		})

		return dto.LoginUserResult{}, application_errors.ErrAccountNotActive
	}

	// Generate the access token only after all authentication checks
	// have succeeded.
	accessToken, err := u.tokenService.GenerateAccessToken(
		ctx,
		authenticatedUser.ID(),
		authenticatedUser.Role(),
	)
	if err != nil {
		u.logAuthenticationEvent(ctx, ports.LogEvent{
			Event:           "auth.login.failed",
			Operation:       "login",
			UserID:          authenticatedUser.ID().String(),
			Role:            authenticatedUser.Role().String(),
			FailureCategory: "token_generation_failed",
		})

		return dto.LoginUserResult{}, application_errors.ErrTokenGeneration
	}

	// Authentication has succeeded.
	//
	// The event deliberately contains only the user's identity and role.
	// The access token and supplied password are never included.
	u.logAuthenticationEvent(ctx, ports.LogEvent{
		Event:     "auth.login.succeeded",
		Operation: "login",
		UserID:    authenticatedUser.ID().String(),
		Role:      authenticatedUser.Role().String(),
	})

	// Return the authenticated identity and generated token.
	return dto.NewLoginUserResult(
		authenticatedUser,
		accessToken,
	), nil
}

// logAuthenticationEvent records an authentication event through the
// application logging port.
//
// Logging is intentionally best-effort. Authentication correctness must
// not depend on the availability of the observability infrastructure.
// A logger failure therefore does not alter the authentication result.
func (u *AuthenticateUserUseCase) logAuthenticationEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if u.logger == nil {
		return
	}

	_ = u.logger.Log(ctx, event)
}

// Compile-time assertion.
//
// This guarantees that the use case satisfies the authentication
// application contract.
var _ ports.AuthenticateUserService = (*AuthenticateUserUseCase)(nil)
