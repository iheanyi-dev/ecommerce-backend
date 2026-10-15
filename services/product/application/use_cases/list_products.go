// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

const (
	SortNewest    = "newest"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortNameAsc   = "name_asc"
)

var ErrInvalidProductFilters = errors.New("invalid product browse filters")

// ListProductsInput contains marketplace browse filters, an optional text query, and pagination.
type ListProductsInput struct {
	Query       string
	MinPrice    *valueobjects.Money
	MaxPrice    *valueobjects.Money
	InStockOnly bool
	SortBy      string
	Limit       int
	Offset      int
}

// ListProducts browses products and removes products from inactive stores.
// It does not fetch replacement products when filtering shortens the page.
type ListProducts struct {
	products ports.ProductRepository
	store    ports.StoreClient
	logger   ports.Logger
	metrics  ports.Metrics
}

func NewListProducts(
	products ports.ProductRepository,
	store ports.StoreClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *ListProducts {
	return &ListProducts{products: products, store: store, logger: logger, metrics: metrics}
}

func (uc *ListProducts) Execute(ctx context.Context, input ListProductsInput) (ProductPage, error) {
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		uc.record(ctx, "product.list.validation_failed", "validation")
		return ProductPage{}, err
	}

	if input.SortBy == "" {
		input.SortBy = SortNewest
	}
	switch input.SortBy {
	case SortNewest, SortPriceAsc, SortPriceDesc, SortNameAsc:
	default:
		uc.record(ctx, "product.list.validation_failed", "validation")
		return ProductPage{}, fmt.Errorf("%w: unsupported sort order", ErrInvalidProductFilters)
	}

	if input.MinPrice != nil && input.MaxPrice != nil {
		if input.MinPrice.Currency() != input.MaxPrice.Currency() ||
			input.MinPrice.Amount() > input.MaxPrice.Amount() {
			uc.record(ctx, "product.list.validation_failed", "validation")
			return ProductPage{}, fmt.Errorf("%w: invalid price range", ErrInvalidProductFilters)
		}
	}

	candidates, total, err := uc.products.ListProducts(ctx, ports.ProductListFilters{
		Query:    strings.TrimSpace(input.Query),
		MinPrice: input.MinPrice, MaxPrice: input.MaxPrice,
		InStockOnly: input.InStockOnly, SortBy: input.SortBy,
	}, limit, offset)
	if err != nil {
		uc.record(ctx, "product.list.repository_failed", "persistence")
		return ProductPage{}, fmt.Errorf("list marketplace products: %w", err)
	}

	storeIDs := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, product := range candidates {
		if product == nil {
			continue
		}
		id := product.StoreID()
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			storeIDs = append(storeIDs, id)
		}
	}

	activeIDs := make(map[string]struct{}, len(storeIDs))
	if len(storeIDs) > 0 {
		active, filterErr := uc.store.FilterActiveStoreIDs(ctx, storeIDs)
		if filterErr != nil {
			uc.record(ctx, "product.list.store_filter_failed", "dependency")
			return ProductPage{}, fmt.Errorf("filter active stores: %w", filterErr)
		}
		for _, id := range active {
			activeIDs[id] = struct{}{}
		}
	}

	filtered := make([]*entities.Product, 0, len(candidates))
	for _, product := range candidates {
		if product == nil {
			continue
		}
		if _, active := activeIDs[product.StoreID()]; active {
			filtered = append(filtered, product)
		}
	}

	uc.record(ctx, "product.list.succeeded", "")
	return ProductPage{
		Products: filtered,
		// This is the repository's candidate total, before active-store filtering.
		// An accurate eligible total requires a separate infrastructure query.
		Total: total, Limit: limit, Offset: offset,
	}, nil
}

func (uc *ListProducts) record(ctx context.Context, event, failure string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "list_products", FailureCategory: failure,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name: "product_operations_total", Value: 1,
			Labels: map[string]string{"operation": "list_products", "event": event},
		})
	}
}
