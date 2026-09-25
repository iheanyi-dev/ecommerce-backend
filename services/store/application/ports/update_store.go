package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// UpdateStoreService updates the profile fields of a Store owned by the
// authenticated user.
type UpdateStoreService interface {
	Execute(ctx context.Context, input dto.UpdateStoreInput) (dto.UpdateStoreOutput, error)
}
