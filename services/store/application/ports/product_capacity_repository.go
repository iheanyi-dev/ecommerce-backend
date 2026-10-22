package ports

import (
	"context"

	"github.com/google/uuid"
)

// ProductCapacityRepository atomically reserves and releases product slots.
type ProductCapacityRepository interface {
	IncrementProductCount(ctx context.Context, storeID uuid.UUID) (int, error)
	DecrementProductCount(ctx context.Context, storeID uuid.UUID) (int, error)
}

// ActiveStoreLookupRepository filters candidate Store IDs by active status.
type ActiveStoreLookupRepository interface {
	FilterActiveStoreIDs(ctx context.Context, storeIDs []uuid.UUID) ([]uuid.UUID, error)
}
