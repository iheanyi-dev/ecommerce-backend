package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// SubscriptionRepository defines persistence operations for store subscriptions.
//
// GetCurrentByStore returns the store's active or pending subscription.
// Create must enforce the one-active-or-pending-subscription-per-store
// invariant atomically. Implementations should translate the corresponding
// database uniqueness conflict to application.ErrSubscriptionConflict.
type SubscriptionRepository interface {
	GetByStoreID(ctx context.Context, storeID uuid.UUID) ([]entities.Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Subscription, error)
	GetCurrentByStore(
		ctx context.Context,
		storeID uuid.UUID,
	) (*entities.Subscription, error)

	Create(ctx context.Context, subscription entities.Subscription) error

	Update(ctx context.Context, subscription entities.Subscription) error
	GetDueForUpdate(ctx context.Context, now time.Time, limit int32) ([]entities.Subscription, error)
}
