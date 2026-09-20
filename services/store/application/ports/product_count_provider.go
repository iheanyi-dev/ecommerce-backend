package ports

import (
	"context"

	"github.com/google/uuid"
)

// ProductCountProvider provides the current number of products belonging to
// a Store without exposing Product Service persistence details to the Store
// domain.
type ProductCountProvider interface {
	CountByStoreID(ctx context.Context, storeID uuid.UUID) (int, error)
}
