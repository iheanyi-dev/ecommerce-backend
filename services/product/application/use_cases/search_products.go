// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"errors"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

// ErrInvalidMarketplaceProductSearch indicates an invalid customer search query.
var ErrInvalidMarketplaceProductSearch = errors.New("invalid marketplace product search")

// SearchProductsInput contains customer-facing marketplace search filters.
type SearchProductsInput struct {
	Query       string
	MinPrice    *valueobjects.Money
	MaxPrice    *valueobjects.Money
	InStockOnly bool
	SortBy      string
	Limit       int
	Offset      int
}

// SearchProducts searches marketplace products for customers.
// Store eligibility filtering and pagination are delegated to the existing
// marketplace listing workflow; the repository receives the text query.
type SearchProducts struct {
	listing *ListProducts
}

// NewSearchProducts constructs the customer-facing marketplace search.
func NewSearchProducts(
	products ports.ProductRepository,
	store ports.StoreClient,
	logger ports.Logger,
	metrics ports.Metrics,
	promotions ports.ProductPromotionRepository,
) *SearchProducts {
	return &SearchProducts{
		listing: NewListProducts(products, store, logger, metrics, promotions),
	}
}

// Execute searches products by text and applies the marketplace filters.
func (uc *SearchProducts) Execute(
	ctx context.Context,
	input SearchProductsInput,
) (ProductPage, error) {
	input.Query = strings.TrimSpace(input.Query)
	if input.Query == "" {
		return ProductPage{}, ErrInvalidMarketplaceProductSearch
	}

	return uc.listing.Execute(ctx, ListProductsInput{
		Query:       input.Query,
		MinPrice:    input.MinPrice,
		MaxPrice:    input.MaxPrice,
		InStockOnly: input.InStockOnly,
		SortBy:      input.SortBy,
		Limit:       input.Limit,
		Offset:      input.Offset,
	})
}
