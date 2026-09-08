package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
)

// ListUsersUseCase retrieves users for administrative user management.
//
// The use case is responsible for:
//   - validating pagination input
//   - retrieving users through the repository abstraction
//   - mapping domain users into safe application DTOs
//
// Sensitive authentication information such as password hashes is never
// included in the returned DTO.
type ListUsersUseCase struct {
	userRepository ports.UserRepository
}

// NewListUsersUseCase creates a new ListUsersUseCase.
func NewListUsersUseCase(
	userRepository ports.UserRepository,
) *ListUsersUseCase {
	return &ListUsersUseCase{
		userRepository: userRepository,
	}
}

// Execute retrieves a paginated list of users.
//
// limit controls the maximum number of users returned.
//
// offset controls how many users are skipped before collecting results.
func (uc *ListUsersUseCase) Execute(
	ctx context.Context,
	limit int,
	offset int,
) (*dto.ListUsersResult, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	if offset < 0 {
		return nil, fmt.Errorf("offset cannot be negative")
	}

	users, err := uc.userRepository.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	result := &dto.ListUsersResult{
		Users: make([]dto.UserSummary, 0, len(users)),
	}

	for _, existingUser := range users {
		result.Users = append(result.Users, dto.UserSummary{
			ID:        existingUser.ID().String(),
			FullName:  existingUser.FullName().String(),
			Email:     existingUser.Email().String(),
			Role:      existingUser.Role().String(),
			Status:    existingUser.Status().String(),
			CreatedAt: existingUser.CreatedAt(),
			UpdatedAt: existingUser.UpdatedAt(),
		})
	}

	return result, nil
}

var _ ports.ListUsersService = (*ListUsersUseCase)(nil)
