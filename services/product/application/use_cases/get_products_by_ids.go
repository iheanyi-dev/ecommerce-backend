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

// ErrInvalidProductIDs indicates that a bulk product lookup received no IDs
// or at least one blank ID.
var ErrInvalidProductIDs = errors.New("invalid product IDs")

// GetProductsByIDs retrieves multiple products for internal consumers such as
// cart and checkout. It does not enforce public store-availability rules.
type GetProductsByIDs struct {
	products   ports.ProductRepository
	promotions ports.ProductPromotionRepository
	logger     ports.Logger
	metrics    ports.Metrics
}

// NewGetProductsByIDs constructs the bulk product retrieval use case.
func NewGetProductsByIDs(
	products ports.ProductRepository,
	logger ports.Logger,
	metrics ports.Metrics,
	promotions ports.ProductPromotionRepository,
) *GetProductsByIDs {
	return &GetProductsByIDs{
		products:   products,
		promotions: promotions,
		logger:     logger,
		metrics:    metrics,
	}
}

// Execute retrieves all requested products in one repository operation.
//
// Results follow the requested ID order, including repeated IDs. If any
// requested product is missing, the operation fails with ErrProductNotFound
// rather than returning a partial result.
func (uc *GetProductsByIDs) Execute(
	ctx context.Context,
	productIDs []string,
) ([]ProductResult, error) {
	if len(productIDs) == 0 {
		uc.record(ctx, "product.get_by_ids.validation_failed", "validation")
		return nil, ErrInvalidProductIDs
	}

	for _, productID := range productIDs {
		if strings.TrimSpace(productID) == "" {
			uc.record(ctx, "product.get_by_ids.validation_failed", "validation")
			return nil, ErrInvalidProductIDs
		}
	}

	products, err := uc.products.GetByIDs(ctx, productIDs)
	if err != nil {
		uc.record(ctx, "product.get_by_ids.repository_failed", "persistence")
		return nil, fmt.Errorf("get products by IDs: %w", err)
	}

	// Index results by ID so the application contract does not depend on
	// repository ordering. Repeated requested IDs remain repeated in output.
	byID := make(map[string]*entities.Product, len(products))
	for _, product := range products {
		if product != nil {
			byID[product.ID()] = product
		}
	}

	ordered := make([]*entities.Product, 0, len(productIDs))
	for _, productID := range productIDs {
		product, found := byID[productID]
		if !found {
			uc.record(ctx, "product.get_by_ids.not_found", "")
			return nil, ErrProductNotFound
		}
		ordered = append(ordered, product)
	}

	results, err := loadProductResults(ctx, uc.promotions, ordered)
	if err != nil {
		uc.record(ctx, "product.get_by_ids.promotion_lookup_failed", "dependency")
		return nil, err
	}

	uc.record(ctx, "product.get_by_ids.succeeded", "")
	return results, nil
}

// record keeps logging and metrics best-effort. Labels exclude product IDs.
func (uc *GetProductsByIDs) record(
	ctx context.Context,
	event string,
	failureCategory string,
) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event:           event,
			Operation:       "get_products_by_ids",
			FailureCategory: failureCategory,
		})
	}

	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "get_products_by_ids",
				"event":     event,
			},
		})
	}
}
