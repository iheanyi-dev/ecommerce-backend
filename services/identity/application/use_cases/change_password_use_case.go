package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/policies"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// ChangePasswordUseCase handles changing an authenticated user's password.
//
// The use case deliberately keeps password handling inside the application
// boundary. Plaintext passwords are only used for verification and hashing;
// they are never passed to the repository or logger.
//
// Logging is best-effort. An observability failure must never change the
// business result of a password-change operation.
type ChangePasswordUseCase struct {
	userRepository ports.UserRepository
	passwordHasher ports.PasswordHasher
	logger         ports.Logger
}

// NewChangePasswordUseCase creates a new ChangePasswordUseCase.
func NewChangePasswordUseCase(
	userRepository ports.UserRepository,
	passwordHasher ports.PasswordHasher,
	logger ports.Logger,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		logger:         logger,
	}
}

// logChangePasswordEvent writes a structured password-change event.
//
// Logging is intentionally best-effort:
//
//   - a nil logger is allowed so the use case remains safe in tests or
//     environments where observability is not configured;
//   - logger failures are deliberately ignored because logging must not
//     alter password-change behavior.
//
// No password or password hash is ever included in the event.
func (u *ChangePasswordUseCase) logChangePasswordEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if u.logger == nil {
		return
	}

	_ = u.logger.Log(ctx, event)
}

// Execute changes the authenticated user's password.
//
// The user ID, current password, and new password are supplied separately
// because this is the existing application service contract. The current
// password is verified against the persisted password hash before the new
// password is accepted.
func (u *ChangePasswordUseCase) Execute(
	ctx context.Context,
	userID string,
	currentPassword string,
	newPassword string,
) error {
	id, err := user.UserIDFromString(userID)
	if err != nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.validation_failed",
			Operation:       "change_password",
			FailureCategory: "validation_failed",
		})

		return err
	}

	existingUser, err := u.userRepository.FindByID(ctx, id)
	if err != nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.failed",
			Operation:       "change_password",
			UserID:          id.String(),
			FailureCategory: "user_lookup_failed",
		})

		return err
	}

	if existingUser == nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.not_found",
			Operation:       "change_password",
			UserID:          id.String(),
			FailureCategory: "user_not_found",
		})

		return errors.ErrUserNotFound
	}

	if existingUser.Status() != user.StatusActive {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.user_inactive",
			Operation:       "change_password",
			UserID:          existingUser.ID().String(),
			Role:            existingUser.Role().String(),
			FailureCategory: "user_inactive",
		})

		return errors.ErrAccountNotActive
	}

	// Verify the supplied current password against the password hash stored
	// on the user. The plaintext password never leaves the password hasher.
	if err := u.passwordHasher.Verify(
		ctx,
		currentPassword,
		existingUser.PasswordHash().String(),
	); err != nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.invalid_credentials",
			Operation:       "change_password",
			UserID:          existingUser.ID().String(),
			Role:            existingUser.Role().String(),
			FailureCategory: "invalid_credentials",
		})

		return err
	}

	// Validate the new password before performing the potentially expensive
	// password-hashing operation.
	if err := policies.ValidatePassword(newPassword); err != nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.validation_failed",
			Operation:       "change_password",
			UserID:          existingUser.ID().String(),
			Role:            existingUser.Role().String(),
			FailureCategory: "validation_failed",
		})

		return err
	}

	// Hash the new password. Only the resulting hash is allowed to reach the
	// domain aggregate and persistence layer.
	newPasswordHash, err := u.passwordHasher.Hash(ctx, newPassword)
	if err != nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.failed",
			Operation:       "change_password",
			UserID:          existingUser.ID().String(),
			Role:            existingUser.Role().String(),
			FailureCategory: "password_hashing_failed",
		})

		return err
	}

	newPasswordHashVO, err := user.NewPasswordHash(newPasswordHash)
	if err != nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.validation_failed",
			Operation:       "change_password",
			UserID:          existingUser.ID().String(),
			Role:            existingUser.Role().String(),
			FailureCategory: "validation_failed",
		})

		return err
	}

	// ChangePassword performs the domain state mutation and updates the
	// aggregate's UpdatedAt timestamp.
	existingUser.ChangePassword(newPasswordHashVO)

	// Persist the complete aggregate so the infrastructure layer stores both
	// the new password hash and the domain-generated UpdatedAt timestamp.
	if err := u.userRepository.UpdatePasswordHash(ctx, existingUser); err != nil {
		u.logChangePasswordEvent(ctx, ports.LogEvent{
			Event:           "auth.password_change.failed",
			Operation:       "change_password",
			UserID:          existingUser.ID().String(),
			Role:            existingUser.Role().String(),
			FailureCategory: "password_update_failed",
		})

		return err
	}

	u.logChangePasswordEvent(ctx, ports.LogEvent{
		Event:     "auth.password_change.succeeded",
		Operation: "change_password",
		UserID:    existingUser.ID().String(),
		Role:      existingUser.Role().String(),
	})

	return nil
}
