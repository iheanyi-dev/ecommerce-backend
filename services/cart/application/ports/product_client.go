package ports

import (
	"context"

	"github.com/google/uuid"
)

// ProductSnapshot contains authoritative product data needed to refresh a
// cart during checkout. It intentionally excludes product images.
//
// Quantity is the current stock quantity in the Product service, not the
// customer's requested cart quantity.
type ProductSnapshot struct {
	ID          uuid.UUID
	StoreID     uuid.UUID
	Name        string
	Description string
	PriceMinor  int64
	Currency    string
	Quantity    int64
}

// ProductClient retrieves current product information from the Product
// service.
type ProductClient interface {
	// GetProductsByIDs returns authoritative snapshots for existing requested
	// products. On a successful call, requested IDs omitted from the response
	// are considered products that no longer exist. Service/transport failures
	// must be returned as errors, never represented as omitted products.
	GetProductsByIDs(ctx context.Context, productIDs []uuid.UUID) ([]ProductSnapshot, error)
}
