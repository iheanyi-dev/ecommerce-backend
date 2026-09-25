package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// ChangeStoreImageService replaces the image associated with a Store owned
// by the authenticated user.
type ChangeStoreImageService interface {
	Execute(ctx context.Context, input dto.ChangeStoreImageInput) (dto.ChangeStoreImageOutput, error)
}
