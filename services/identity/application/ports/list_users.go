package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
)

// ListUsersService defines the application operation used to retrieve
// a paginated list of users for administrative user management.
type ListUsersService interface {
	Execute(
		ctx context.Context,
		limit int,
		offset int,
	) (*dto.ListUsersResult, error)
}
