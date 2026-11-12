package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

// ProductPromotionRepository persists promotion availability for stores and products.
type ProductPromotionRepository interface {
	Add(ctx context.Context, promotion *entities.ProductPromotion) error
	Update(ctx context.Context, promotion *entities.ProductPromotion) error
	Remove(ctx context.Context, storeID, promotionID string) error
	Exists(ctx context.Context, storeID string, productID *string) (bool, error)
}
