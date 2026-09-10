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
//   - recording structured observability events without exposing secrets
//
// Sensitive authentication information such as password hashes is never
// included in the returned DTO or observability events.
type ListUsersUseCase struct {
	userRepository ports.UserRepository
	logger         ports.Logger
}

// NewListUsersUseCase creates a new ListUsersUseCase.
func NewListUsersUseCase(
	userRepository ports.UserRepository,
	logger ports.Logger,
) *ListUsersUseCase {
	return &ListUsersUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// logListUsersEvent records an observability event on a best-effort basis.
//
// Logging must never change the business result of the use case. Therefore,
// logger failures are intentionally ignored.
func (uc *ListUsersUseCase) logListUsersEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
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
		uc.logListUsersEvent(ctx, ports.LogEvent{
			Event:           "user.list.validation_failed",
			Operation:       "list_users",
			FailureCategory: "validation_failed",
		})

		return nil, fmt.Errorf("limit must be greater than zero")
	}

	if offset < 0 {
		uc.logListUsersEvent(ctx, ports.LogEvent{
			Event:           "user.list.validation_failed",
			Operation:       "list_users",
			FailureCategory: "validation_failed",
		})

		return nil, fmt.Errorf("offset cannot be negative")
	}

	users, err := uc.userRepository.List(
		ctx,
		limit,
		offset,
	)
	if err != nil {
		uc.logListUsersEvent(ctx, ports.LogEvent{
			Event:           "user.list.failed",
			Operation:       "list_users",
			FailureCategory: "user_lookup_failed",
		})

		return nil, fmt.Errorf(
			"failed to list users: %w",
			err,
		)
	}

	result := &dto.ListUsersResult{
		Users: make([]dto.UserSummary, 0, len(users)),
	}

	for _, existingUser := range users {
		result.Users = append(
			result.Users,
			dto.UserSummary{
				ID:        existingUser.ID().String(),
				FullName:  existingUser.FullName().String(),
				Email:     existingUser.Email().String(),
				Role:      existingUser.Role().String(),
				Status:    existingUser.Status().String(),
				CreatedAt: existingUser.CreatedAt(),
				UpdatedAt: existingUser.UpdatedAt(),
			},
		)
	}

	uc.logListUsersEvent(ctx, ports.LogEvent{
		Event:     "user.list.succeeded",
		Operation: "list_users",
	})

	return result, nil
}

var _ ports.ListUsersService = (*ListUsersUseCase)(nil)
