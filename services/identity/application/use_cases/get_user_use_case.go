package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
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
//   - emitting structured observability events
//
// Sensitive authentication information such as PasswordHash is never
// included in the returned DTO or observability events.
type GetUserUseCase struct {
	userRepository ports.UserRepository
	logger         ports.Logger
}

// NewGetUserUseCase creates a new GetUserUseCase.
func NewGetUserUseCase(
	userRepository ports.UserRepository,
	logger ports.Logger,
) *GetUserUseCase {
	return &GetUserUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// logGetUserEvent emits a structured Get User event.
//
// Logging is intentionally best-effort. Observability must never change the
// business outcome of the Get User operation.
func (uc *GetUserUseCase) logGetUserEvent(
	ctx context.Context,
	event ports.LogEvent,
) {
	if uc.logger == nil {
		return
	}

	_ = uc.logger.Log(ctx, event)
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
		uc.logGetUserEvent(ctx, ports.LogEvent{
			Event:           "user.get.validation_failed",
			Operation:       "get_user",
			FailureCategory: "validation_failed",
		})

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
		uc.logGetUserEvent(ctx, ports.LogEvent{
			Event:           "user.get.failed",
			Operation:       "get_user",
			UserID:          id.String(),
			FailureCategory: "user_lookup_failed",
		})

		return nil, fmt.Errorf(
			"failed to find user: %w",
			err,
		)
	}

	// A nil user with no repository error represents a valid
	// "not found" result at the application boundary.
	if existingUser == nil {
		uc.logGetUserEvent(ctx, ports.LogEvent{
			Event:           "user.get.not_found",
			Operation:       "get_user",
			UserID:          id.String(),
			FailureCategory: "user_not_found",
		})

		return nil, application_errors.ErrUserNotFound
	}

	result := &dto.GetUserResult{
		ID:        existingUser.ID().String(),
		FullName:  existingUser.FullName().String(),
		Email:     existingUser.Email().String(),
		Role:      existingUser.Role().String(),
		Status:    existingUser.Status().String(),
		CreatedAt: existingUser.CreatedAt(),
		UpdatedAt: existingUser.UpdatedAt(),
	}

	uc.logGetUserEvent(ctx, ports.LogEvent{
		Event:     "user.get.succeeded",
		Operation: "get_user",
		UserID:    result.ID,
		Role:      result.Role,
	})

	return result, nil
}

var _ ports.GetUserService = (*GetUserUseCase)(nil)
