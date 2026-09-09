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
type UpdateUserStatusUseCase struct {
	userRepository ports.UserRepository
}

// NewUpdateUserStatusUseCase constructs the administrative account-status
// update use case.
func NewUpdateUserStatusUseCase(
	userRepository ports.UserRepository,
) *UpdateUserStatusUseCase {
	return &UpdateUserStatusUseCase{
		userRepository: userRepository,
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
		return dto.UpdateUserStatusResult{}, err
	}

	existingUser, err := uc.userRepository.FindByID(ctx, id)
	if err != nil {
		return dto.UpdateUserStatusResult{}, err
	}

	if existingUser == nil {
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
		return dto.UpdateUserStatusResult{}, domain_errors.ErrInvalidStatusTransition
	}

	if err != nil {
		return dto.UpdateUserStatusResult{}, err
	}

	if err := uc.userRepository.UpdateStatus(ctx, existingUser); err != nil {
		return dto.UpdateUserStatusResult{}, err
	}

	return dto.NewUpdateUserStatusResult(existingUser), nil
}

var _ ports.UpdateUserStatusService = (*UpdateUserStatusUseCase)(nil)
