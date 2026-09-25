package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// GetStoreByIDService retrieves a Store by its unique identifier.
type GetStoreByIDService interface {
	Execute(ctx context.Context, storeID uuid.UUID) (dto.StoreOutput, error)
}
