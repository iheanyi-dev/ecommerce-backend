package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
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
//   - returning the authenticated user's identity
type AuthenticateUserUseCase struct {
	userRepository ports.UserRepository
	passwordHasher ports.PasswordHasher
	tokenService   ports.TokenService
}

// NewAuthenticateUserUseCase creates the authentication use case.
//
// Dependencies are supplied through application ports, keeping the use
// case independent from PostgreSQL, bcrypt, JWT, HTTP, or infrastructure.
func NewAuthenticateUserUseCase(
	userRepository ports.UserRepository,
	passwordHasher ports.PasswordHasher,
	tokenService ports.TokenService,
) *AuthenticateUserUseCase {
	return &AuthenticateUserUseCase{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		tokenService:   tokenService,
	}
}

// Authenticate verifies credentials, validates the account status,
// and generates an access token for a successful authentication.
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
		return dto.LoginUserResult{}, ErrInvalidCredentials
	}

	// Retrieve the persisted user.
	//
	// A missing user intentionally produces the same generic error used
	// when the password is incorrect.
	authenticatedUser, err := u.userRepository.FindByEmail(ctx, email)
	if err != nil {
		return dto.LoginUserResult{}, ErrInvalidCredentials
	}

	if authenticatedUser == nil {
		return dto.LoginUserResult{}, ErrInvalidCredentials
	}

	// Verify the supplied plaintext password against the stored hash.
	if err := u.passwordHasher.Verify(
		ctx,
		command.Password,
		authenticatedUser.PasswordHash().String(),
	); err != nil {
		return dto.LoginUserResult{}, ErrInvalidCredentials
	}

	// Only active accounts may authenticate.
	if authenticatedUser.Status() != user.StatusActive {
		return dto.LoginUserResult{}, ErrAccountNotActive
	}

	// Generate the access token only after all authentication checks
	// have succeeded.
	accessToken, err := u.tokenService.GenerateAccessToken(
		ctx,
		authenticatedUser.ID(),
		authenticatedUser.Role(),
	)
	if err != nil {
		return dto.LoginUserResult{}, ErrTokenGeneration
	}

	// Return the authenticated identity and generated token.
	return dto.NewLoginUserResult(
		authenticatedUser,
		accessToken,
	), nil
}

// Compile-time assertion.
//
// This guarantees that the use case satisfies the authentication
// application contract.
var _ ports.AuthenticateUserService = (*AuthenticateUserUseCase)(nil)
