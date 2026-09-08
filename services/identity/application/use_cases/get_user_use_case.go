package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
)

// GetUserUseCase retrieves a single user for administrative user
// management.
//
// The use case is responsible for:
//   - validating the supplied user ID
//   - retrieving the user through the repository abstraction
//   - converting the domain aggregate into a safe application DTO
//   - translating a missing user into ErrUserNotFound
//
// Sensitive authentication information such as PasswordHash is never
// included in the returned DTO.
type GetUserUseCase struct {
	userRepository ports.UserRepository
}

// NewGetUserUseCase creates a new GetUserUseCase.
func NewGetUserUseCase(
	userRepository ports.UserRepository,
) *GetUserUseCase {
	return &GetUserUseCase{
		userRepository: userRepository,
	}
}

// Execute retrieves a single user by ID.
//
// userID is accepted as a string because the application boundary may
// receive the value directly from an HTTP path parameter.
//
// The string is converted into the domain UserID before it crosses
// into the repository boundary.
func (uc *GetUserUseCase) Execute(
	ctx context.Context,
	userID string,
) (*dto.GetUserResult, error) {
	id, err := user.UserIDFromString(userID)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid user ID: %w",
			err,
		)
	}

	existingUser, err := uc.userRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to find user: %w",
			err,
		)
	}

	// A nil user with no repository error represents a valid
	// "not found" result at the application boundary.
	if existingUser == nil {
		return nil, ErrUserNotFound
	}

	return &dto.GetUserResult{
		ID:        existingUser.ID().String(),
		FullName:  existingUser.FullName().String(),
		Email:     existingUser.Email().String(),
		Role:      existingUser.Role().String(),
		Status:    existingUser.Status().String(),
		CreatedAt: existingUser.CreatedAt(),
		UpdatedAt: existingUser.UpdatedAt(),
	}, nil
}

var _ ports.GetUserService = (*GetUserUseCase)(nil)
