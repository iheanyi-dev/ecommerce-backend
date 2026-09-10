package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/policies"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// RegisterUserUseCase orchestrates the user registration workflow.
//
// The use case coordinates domain validation, password hashing, and
// persistence while keeping infrastructure concerns outside the application
// layer.
//
// Logging is deliberately best-effort. A logging failure must never cause an
// otherwise successful registration to fail.
type RegisterUserUseCase struct {
	userRepository ports.UserRepository
	passwordHasher ports.PasswordHasher
	logger         ports.Logger
}

// NewRegisterUserUseCase creates a new RegisterUserUseCase.
func NewRegisterUserUseCase(
	userRepository ports.UserRepository,
	passwordHasher ports.PasswordHasher,
	logger ports.Logger,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		logger:         logger,
	}
}

// logRegistrationEvent emits a structured registration event.
//
// Logging is intentionally best-effort. Observability must never change the
// business outcome of the registration operation.
func (u *RegisterUserUseCase) logRegistrationEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if u.logger == nil {
		return
	}

	_ = u.logger.Log(ctx, event)
}

// Execute registers a new user.
func (u *RegisterUserUseCase) Execute(
	ctx context.Context,
	command dto.RegisterUserCommand,
) (dto.RegisterUserResult, error) {
	// Validate and construct the email value object first.
	email, err := user.NewEmail(command.Email)
	if err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.validation_failed",
			Operation:       "registration",
			FailureCategory: "validation_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	// Check whether the email is already registered.
	exists, err := u.userRepository.ExistsByEmail(ctx, email)
	if err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.failed",
			Operation:       "registration",
			FailureCategory: "email_existence_check_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	if exists {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.duplicate_email",
			Operation:       "registration",
			FailureCategory: "duplicate_email",
		})

		return dto.RegisterUserResult{}, errors.ErrEmailAlreadyExists
	}

	// Apply the application's password policy before hashing.
	if err := policies.ValidatePassword(command.Password); err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.validation_failed",
			Operation:       "registration",
			FailureCategory: "validation_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	// Hash the plaintext password. The plaintext password is never logged.
	passwordHash, err := u.passwordHasher.Hash(ctx, command.Password)
	if err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.failed",
			Operation:       "registration",
			FailureCategory: "password_hashing_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	// Construct the password-hash value object.
	hash, err := user.NewPasswordHash(passwordHash)
	if err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.validation_failed",
			Operation:       "registration",
			FailureCategory: "validation_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	// Construct the full-name value object.
	fullName, err := user.NewFullName(command.FullName)
	if err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.validation_failed",
			Operation:       "registration",
			FailureCategory: "validation_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	// Construct the User aggregate.
	newUser, err := user.NewUser(
		fullName,
		email,
		hash,
	)
	if err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.validation_failed",
			Operation:       "registration",
			FailureCategory: "validation_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	// Persist the newly-created user.
	if err := u.userRepository.Create(ctx, newUser); err != nil {
		u.logRegistrationEvent(ctx, ports.LogEvent{
			Event:           "auth.registration.failed",
			Operation:       "registration",
			FailureCategory: "user_creation_failed",
		})

		return dto.RegisterUserResult{}, err
	}

	result := dto.NewRegisterUserResult(newUser)

	// Registration succeeded. Only non-sensitive identity information is
	// included in the structured event.
	u.logRegistrationEvent(ctx, ports.LogEvent{
		Event:     "auth.registration.succeeded",
		Operation: "registration",
		UserID:    result.ID,
		Role:      result.Role,
	})

	return result, nil
}
