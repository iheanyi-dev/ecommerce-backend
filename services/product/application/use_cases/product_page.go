// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

var ErrInvalidPagination = errors.New("invalid pagination")

// ProductResult combines the Product entity with derived promotion availability.
type ProductResult struct {
	*entities.Product
	HasPromotion bool
}

// ProductPage contains a stable page of products and the total number of
// matching products across all pages.
type ProductPage struct {
	Products []ProductResult
	Total    int64
	Limit    int
	Offset   int
}

// loadProductResults enriches products without changing the domain entity.
// Repeated product IDs are checked only once.
func loadProductResults(
	ctx context.Context,
	promotions ports.ProductPromotionRepository,
	products []*entities.Product,
) ([]ProductResult, error) {
	results := make([]ProductResult, 0, len(products))
	checked := make(map[string]bool, len(products))

	for _, product := range products {
		if product == nil {
			continue
		}

		id := product.ID()
		hasPromotion, exists := checked[id]
		if !exists && promotions != nil {
			var err error
			hasPromotion, err = promotions.Exists(ctx, product.StoreID(), &id)
			if err != nil {
				return nil, fmt.Errorf("check product promotion availability: %w", err)
			}
			checked[id] = hasPromotion
		}

		results = append(results, ProductResult{
			Product:      product,
			HasPromotion: hasPromotion,
		})
	}

	return results, nil
}

func normalizePagination(limit, offset int) (int, int, error) {
	if limit < 0 || offset < 0 {
		return 0, 0, ErrInvalidPagination
	}
	if limit == 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return limit, offset, nil
}

func emptyProductPage(limit, offset int) ProductPage {
	return ProductPage{
		Products: make([]ProductResult, 0),
		Total:    0,
		Limit:    limit,
		Offset:   offset,
	}
}

func validIdentifier(value string) bool {
	return strings.TrimSpace(value) != ""
}
