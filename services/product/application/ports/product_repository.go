// Package ports defines the Product application's outbound contracts.
package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

// ProductListFilters describes supported marketplace browse filters.
// Price bounds use Money to preserve currency and precision semantics.
type ProductListFilters struct {
	Query       string
	MinPrice    *valueobjects.Money
	MaxPrice    *valueobjects.Money
	InStockOnly bool
	SortBy      string
}

// ProductRepository persists Product aggregates, including their ordered
// Storage references. Implementations must preserve image ordering.
type StoreProductSearchFilters struct {
	Currency          string
	MinPriceAmount    *int64
	MaxPriceAmount    *int64
	MinPriceInclusive bool
	MaxPriceInclusive bool
	AvailableOnly     bool
}

type ProductRepository interface {
	Create(ctx context.Context, product *entities.Product) error
	GetByID(ctx context.Context, productID string) (*entities.Product, error)
	// GetByIDs retrieves all products matching the supplied IDs in one operation.
	// Missing IDs are omitted; the application layer determines whether any are missing.
	GetByIDs(ctx context.Context, productIDs []string) ([]*entities.Product, error)
	Update(ctx context.Context, product *entities.Product) error
	Delete(ctx context.Context, productID string) error
	// ListByStore returns one page and the total number of matching products.
	ListByStore(ctx context.Context, storeID string, limit, offset int) ([]*entities.Product, int64, error)
	// CountByStore returns the persisted product count for a store.
	CountByStore(ctx context.Context, storeID string) (int64, error)
	// ListProducts returns a candidate page and its pre-eligibility total.
	ListProducts(ctx context.Context, filters ProductListFilters, limit, offset int) ([]*entities.Product, int64, error)

	SearchByStore(
		ctx context.Context,
		storeID string,
		query string,
		filters StoreProductSearchFilters,
	) ([]*entities.Product, error)
}
