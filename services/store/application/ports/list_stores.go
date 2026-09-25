package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// ListStoresService retrieves the publicly discoverable Stores using
// optional search and pagination parameters.
type ListStoresService interface {
	Execute(
		ctx context.Context,
		query string,
		page int,
		pageSize int,
	) (*dto.ListStoresOutput, error)
}
