package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// GetUserService defines the application operation used to retrieve
// a single user for administrative user management.
type GetUserService interface {
	Execute(
		ctx context.Context,
		userID string,
	) (*dto.GetUserResult, error)
}
