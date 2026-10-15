// Package ports defines outbound contracts required by Product application use cases.
package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

// StoreProductSearchFilters contains validated, store-scoped search constraints.
// Price amounts use integer minor units to avoid floating-point comparisons.
type StoreProductSearchFilters struct {
	Currency          string
	MinPriceAmount    *int64
	MaxPriceAmount    *int64
	MinPriceInclusive bool
	MaxPriceInclusive bool
	AvailableOnly     bool
}

// StoreProductSearchRepository searches product names and descriptions within one
// store and applies price and stock filters in persistence.
type StoreProductSearchRepository interface {
	SearchByStore(
		ctx context.Context,
		storeID string,
		query string,
		filters StoreProductSearchFilters,
	) ([]*entities.Product, error)
}
