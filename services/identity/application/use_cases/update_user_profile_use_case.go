package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// UpdateUserProfileUseCase implements the authenticated user's self-service
// profile update workflow.
//
// The use case deliberately works with the authenticated user's ID rather
// than accepting a user ID from the request body. This guarantees that a
// caller can only modify their own profile.
type UpdateUserProfileUseCase struct {
	userRepository ports.UserRepository
	logger         ports.Logger
}

// NewUpdateUserProfileUseCase constructs the profile update use case.
func NewUpdateUserProfileUseCase(
	userRepository ports.UserRepository,
	logger ports.Logger,
) *UpdateUserProfileUseCase {
	return &UpdateUserProfileUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// logUpdateUserProfileEvent records an observability event on a best-effort
// basis.
//
// Logging must never change the business result of the use case. Therefore,
// logger failures are intentionally ignored.
func (uc *UpdateUserProfileUseCase) logUpdateUserProfileEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
}

// Execute updates the authenticated user's full name.
//
// The user ID comes from the authenticated request context, not from the
// HTTP request body. This prevents a caller from selecting another user's
// account through the profile endpoint.
func (uc *UpdateUserProfileUseCase) Execute(
	ctx context.Context,
	userID string,
	command dto.UpdateUserProfileCommand,
) (dto.UpdateUserProfileResult, error) {
	id, err := user.UserIDFromString(userID)
	if err != nil {
		uc.logUpdateUserProfileEvent(ctx, ports.LogEvent{
			Event:           "user.profile_update.validation_failed",
			Operation:       "update_user_profile",
			FailureCategory: "validation_failed",
		})

		return dto.UpdateUserProfileResult{}, err
	}

	fullName, err := user.NewFullName(command.FullName)
	if err != nil {
		uc.logUpdateUserProfileEvent(ctx, ports.LogEvent{
			Event:           "user.profile_update.validation_failed",
			Operation:       "update_user_profile",
			UserID:          id.String(),
			FailureCategory: "validation_failed",
		})

		return dto.UpdateUserProfileResult{}, err
	}

	existingUser, err := uc.userRepository.FindByID(ctx, id)
	if err != nil {
		uc.logUpdateUserProfileEvent(ctx, ports.LogEvent{
			Event:           "user.profile_update.failed",
			Operation:       "update_user_profile",
			UserID:          id.String(),
			FailureCategory: "user_lookup_failed",
		})

		return dto.UpdateUserProfileResult{}, err
	}

	if existingUser == nil {
		uc.logUpdateUserProfileEvent(ctx, ports.LogEvent{
			Event:           "user.profile_update.not_found",
			Operation:       "update_user_profile",
			UserID:          id.String(),
			FailureCategory: "user_not_found",
		})

		return dto.UpdateUserProfileResult{}, application_errors.ErrUserNotFound
	}

	existingUser.ChangeFullName(fullName)

	if err := uc.userRepository.UpdateFullName(
		ctx,
		existingUser,
	); err != nil {
		uc.logUpdateUserProfileEvent(ctx, ports.LogEvent{
			Event:           "user.profile_update.failed",
			Operation:       "update_user_profile",
			UserID:          id.String(),
			Role:            existingUser.Role().String(),
			FailureCategory: "profile_update_failed",
		})

		return dto.UpdateUserProfileResult{}, err
	}

	result := dto.NewUpdateUserProfileResult(existingUser)

	uc.logUpdateUserProfileEvent(ctx, ports.LogEvent{
		Event:     "user.profile_update.succeeded",
		Operation: "update_user_profile",
		UserID:    result.UserID,
		Role:      existingUser.Role().String(),
	})

	return result, nil
}

// Compile-time assertion guarantees that the use case satisfies the
// application service contract.
var _ ports.UpdateUserProfileService = (*UpdateUserProfileUseCase)(nil)
