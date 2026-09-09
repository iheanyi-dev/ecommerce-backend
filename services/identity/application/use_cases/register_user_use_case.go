package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/policies"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// RegisterUserUseCase handles the application workflow for registering
// a new user.
//
// The use case coordinates the domain and application ports, but does not
// know how users are persisted or how passwords are hashed.
//
// Business rules remain inside the domain layer.
type RegisterUserUseCase struct {
	userRepository ports.UserRepository
	passwordHasher ports.PasswordHasher
}

// NewRegisterUserUseCase creates a RegisterUserUseCase with its required
// application dependencies.
func NewRegisterUserUseCase(
	userRepository ports.UserRepository,
	passwordHasher ports.PasswordHasher,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
	}
}

// Execute registers a new user.
//
// The registration workflow is:
//
//  1. Validate the email through the Email value object.
//  2. Check whether the email already exists.
//  3. Validate the plaintext password against the application password policy.
//  4. Hash the supplied password.
//  5. Validate the full name.
//  6. Create the User aggregate.
//  7. Persist the User aggregate through the repository.
//
// A Unit of Work is intentionally not used here because registration
// currently performs a single aggregate persistence operation. Transaction
// coordination will be introduced when the service actually requires
// multiple related writes to be atomic.
func (uc *RegisterUserUseCase) Execute(
	ctx context.Context,
	command dto.RegisterUserCommand,
) (dto.RegisterUserResult, error) {
	// Convert the raw email into the domain Email value object.
	//
	// This ensures that invalid email values are rejected before any
	// persistence operation takes place.
	email, err := user.NewEmail(command.Email)
	if err != nil {
		return dto.RegisterUserResult{}, err
	}

	// Check whether another user already owns this email address.
	exists, err := uc.userRepository.ExistsByEmail(ctx, email)
	if err != nil {
		return dto.RegisterUserResult{}, err
	}

	if exists {
		return dto.RegisterUserResult{}, application_errors.ErrEmailAlreadyExists
	}

	// Validate the plaintext password before performing the expensive
	// password-hashing operation.
	//
	// The password policy belongs to the application layer. The domain
	// never receives the plaintext password; it only receives the
	// resulting PasswordHash value below.
	if err := policies.ValidatePassword(command.Password); err != nil {
		return dto.RegisterUserResult{}, err
	}

	// Password hashing is an application concern rather than a domain
	// concern. The use case delegates this operation to the PasswordHasher
	// port.
	passwordHashValue, err := uc.passwordHasher.Hash(
		ctx,
		command.Password,
	)
	if err != nil {
		return dto.RegisterUserResult{}, err
	}

	passwordHash, err := user.NewPasswordHash(passwordHashValue)
	if err != nil {
		return dto.RegisterUserResult{}, err
	}

	// Convert the supplied name into the corresponding domain value object.
	fullName, err := user.NewFullName(command.FullName)
	if err != nil {
		return dto.RegisterUserResult{}, err
	}

	// Create the User aggregate using validated domain value objects.
	newUser, err := user.NewUser(
		fullName,
		email,
		passwordHash,
	)
	if err != nil {
		return dto.RegisterUserResult{}, err
	}

	// Persist the newly created aggregate.
	//
	// The repository method is intentionally called Create rather than Save
	// because registration creates a new user. This keeps the persistence
	// contract explicit and avoids giving Save multiple meanings.
	if err := uc.userRepository.Create(ctx, newUser); err != nil {
		return dto.RegisterUserResult{}, err
	}

	// Convert the domain aggregate into the application result DTO.
	return dto.NewRegisterUserResult(newUser), nil
}

var _ ports.RegisterUserService = (*RegisterUserUseCase)(nil)
