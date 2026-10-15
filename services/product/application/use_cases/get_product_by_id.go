// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

// ErrProductNotFound indicates that a product does not exist or is not
// publicly available. Inactive-store products intentionally use the same
// error to avoid revealing their existence to public callers.
var ErrProductNotFound = errors.New("product not found")

// GetProductByID retrieves a publicly available product by its ID.
type GetProductByID struct {
	products ports.ProductRepository
	store    ports.StoreClient
	logger   ports.Logger
	metrics  ports.Metrics
}

// NewGetProductByID constructs the public product retrieval use case.
func NewGetProductByID(
	products ports.ProductRepository,
	store ports.StoreClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetProductByID {
	return &GetProductByID{
		products: products,
		store:    store,
		logger:   logger,
		metrics:  metrics,
	}
}

// Execute retrieves the product and verifies that its owning store is active.
// Observability is best-effort and cannot change the business result.
func (uc *GetProductByID) Execute(
	ctx context.Context,
	productID string,
) (*entities.Product, error) {
	product, err := uc.products.GetByID(ctx, productID)
	if err != nil {
		uc.record(ctx, "product.get_by_id.repository_failed", "persistence")
		return nil, fmt.Errorf("get product by ID: %w", err)
	}

	if product == nil {
		uc.record(ctx, "product.get_by_id.not_found", "")
		return nil, ErrProductNotFound
	}

	active, err := uc.store.IsStoreActive(ctx, product.StoreID())
	if err != nil {
		uc.record(ctx, "product.get_by_id.store_check_failed", "store")
		return nil, fmt.Errorf("check product store activity: %w", err)
	}

	if !active {
		uc.record(ctx, "product.get_by_id.store_inactive", "")
		return nil, ErrProductNotFound
	}

	uc.record(ctx, "product.get_by_id.succeeded", "")
	return product, nil
}

// record keeps logging and metrics best-effort. Labels deliberately exclude
// product/store identifiers and other high-cardinality or sensitive values.
func (uc *GetProductByID) record(
	ctx context.Context,
	event string,
	failureCategory string,
) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event:           event,
			Operation:       "get_product_by_id",
			FailureCategory: failureCategory,
		})
	}

	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "get_product_by_id",
				"event":     event,
			},
		})
	}
}
