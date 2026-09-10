package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// UpdateUserStatusUseCase implements the administrator's account-status
// management workflow.
//
// This use case deliberately changes only the account status. Profile fields,
// password, email, and role are outside the responsibility of this workflow.
//
// Logging is best-effort observability. A logging failure must never change
// the business result of the status update operation.
type UpdateUserStatusUseCase struct {
	userRepository ports.UserRepository
	logger         ports.Logger
}

// NewUpdateUserStatusUseCase constructs the administrative account-status
// update use case.
func NewUpdateUserStatusUseCase(
	userRepository ports.UserRepository,
	logger ports.Logger,
) *UpdateUserStatusUseCase {
	return &UpdateUserStatusUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// Execute changes the target user's account status.
//
// The requested transition is delegated to the User aggregate so that domain
// lifecycle rules remain enforced in one place.
func (uc *UpdateUserStatusUseCase) Execute(
	ctx context.Context,
	userID string,
	command dto.UpdateUserStatusCommand,
) (dto.UpdateUserStatusResult, error) {
	id, err := user.UserIDFromString(userID)
	if err != nil {
		uc.logStatusUpdateEvent(ctx, ports.LogEvent{
			Event:           "user.status_update.validation_failed",
			Operation:       "update_user_status",
			FailureCategory: "validation_failed",
		})

		return dto.UpdateUserStatusResult{}, err
	}

	existingUser, err := uc.userRepository.FindByID(ctx, id)
	if err != nil {
		uc.logStatusUpdateEvent(ctx, ports.LogEvent{
			Event:           "user.status_update.failed",
			Operation:       "update_user_status",
			UserID:          id.String(),
			FailureCategory: "user_lookup_failed",
		})

		return dto.UpdateUserStatusResult{}, err
	}

	if existingUser == nil {
		uc.logStatusUpdateEvent(ctx, ports.LogEvent{
			Event:           "user.status_update.not_found",
			Operation:       "update_user_status",
			UserID:          id.String(),
			FailureCategory: "user_not_found",
		})

		return dto.UpdateUserStatusResult{}, application_errors.ErrUserNotFound
	}

	switch user.Status(command.Status) {
	case user.StatusActive:
		err = existingUser.Activate()

	case user.StatusSuspended:
		err = existingUser.Suspend()

	case user.StatusInactive:
		err = existingUser.Deactivate()

	default:
		uc.logStatusUpdateEvent(ctx, ports.LogEvent{
			Event:           "user.status_update.validation_failed",
			Operation:       "update_user_status",
			UserID:          existingUser.ID().String(),
			Role:            string(existingUser.Role()),
			FailureCategory: "validation_failed",
		})

		return dto.UpdateUserStatusResult{}, domain_errors.ErrInvalidStatusTransition
	}

	if err != nil {
		uc.logStatusUpdateEvent(ctx, ports.LogEvent{
			Event:           "user.status_update.validation_failed",
			Operation:       "update_user_status",
			UserID:          existingUser.ID().String(),
			Role:            string(existingUser.Role()),
			FailureCategory: "validation_failed",
		})

		return dto.UpdateUserStatusResult{}, err
	}

	if err := uc.userRepository.UpdateStatus(ctx, existingUser); err != nil {
		uc.logStatusUpdateEvent(ctx, ports.LogEvent{
			Event:           "user.status_update.failed",
			Operation:       "update_user_status",
			UserID:          existingUser.ID().String(),
			Role:            string(existingUser.Role()),
			FailureCategory: "status_update_failed",
		})

		return dto.UpdateUserStatusResult{}, err
	}

	uc.logStatusUpdateEvent(ctx, ports.LogEvent{
		Event:     "user.status_update.succeeded",
		Operation: "update_user_status",
		UserID:    existingUser.ID().String(),
		Role:      string(existingUser.Role()),
	})

	return dto.NewUpdateUserStatusResult(existingUser), nil
}

// logStatusUpdateEvent records a status-management event without allowing
// observability failures to affect the application workflow.
//
// Logger errors are deliberately ignored because logging is supplementary
// infrastructure and must not cause an otherwise successful business
// operation to fail.
func (uc *UpdateUserStatusUseCase) logStatusUpdateEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
}

var _ ports.UpdateUserStatusService = (*UpdateUserStatusUseCase)(nil)
