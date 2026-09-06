package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// UpdateUserProfileUseCase handles self-service updates to a user's
// mutable profile information.
//
// Phase 7 intentionally allows only the full name to be changed here.
// Email remains immutable.
type UpdateUserProfileUseCase struct {
	userRepository ports.UserRepository
}

// NewUpdateUserProfileUseCase creates the profile-update use case.
func NewUpdateUserProfileUseCase(
	userRepository ports.UserRepository,
) *UpdateUserProfileUseCase {
	return &UpdateUserProfileUseCase{
		userRepository: userRepository,
	}
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
	// Convert the authenticated identity into the domain UserID value object.
	id, err := user.UserIDFromString(userID)
	if err != nil {
		return dto.UpdateUserProfileResult{}, err
	}

	// Validate the new full name through the domain value object.
	fullName, err := user.NewFullName(command.FullName)
	if err != nil {
		return dto.UpdateUserProfileResult{}, err
	}

	// Load the current aggregate so the response represents the persisted
	// account and so immutable fields such as email remain untouched.
	existingUser, err := uc.userRepository.FindByID(ctx, id)
	if err != nil {
		return dto.UpdateUserProfileResult{}, err
	}

	if existingUser == nil {
		return dto.UpdateUserProfileResult{}, ErrUserNotFound
	}

	// Apply the domain behaviour rather than directly changing persistence
	// fields. This also updates the aggregate's UpdatedAt timestamp.
	existingUser.ChangeFullName(fullName)

	// Persist only the permitted profile change.
	if err := uc.userRepository.UpdateFullName(
		ctx,
		id,
		existingUser.FullName(),
	); err != nil {
		return dto.UpdateUserProfileResult{}, err
	}

	return dto.NewUpdateUserProfileResult(existingUser), nil
}

// Compile-time assertion guarantees that the use case satisfies the
// application service contract.
var _ ports.UpdateUserProfileService = (*UpdateUserProfileUseCase)(nil)
