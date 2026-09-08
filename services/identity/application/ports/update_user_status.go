package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// UpdateUserStatusService defines the application boundary for an
// administrator changing a user's account status.
type UpdateUserStatusService interface {
	Execute(
		ctx context.Context,
		userID string,
		command dto.UpdateUserStatusCommand,
	) (dto.UpdateUserStatusResult, error)
}
