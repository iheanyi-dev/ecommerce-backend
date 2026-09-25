package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// GetStoreBySlugService retrieves a Store using its public slug.
type GetStoreBySlugService interface {
	Execute(ctx context.Context, slug string) (*dto.StoreOutput, error)
}
