package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// PayoutRepository persists payouts and supports business-level idempotency.
type PayoutRepository interface {
	GetByID(ctx context.Context, id string) (*entities.Payout, error)
	GetByBusinessReference(ctx context.Context, reference value_objects.BusinessReference) (*entities.Payout, error)
	ListByStoreID(ctx context.Context, storeID string, limit, offset int) ([]*entities.Payout, int64, error)
	Save(ctx context.Context, payout *entities.Payout) error
}
