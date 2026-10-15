// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

// GetMyProductByID retrieves a product only when it belongs to the
// authenticated vendor's store.
type GetMyProductByID struct {
	products ports.ProductRepository
	store    ports.StoreClient
	logger   ports.Logger
	metrics  ports.Metrics
}

// NewGetMyProductByID constructs the vendor-owned product retrieval use case.
func NewGetMyProductByID(
	products ports.ProductRepository,
	store ports.StoreClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetMyProductByID {
	return &GetMyProductByID{
		products: products,
		store:    store,
		logger:   logger,
		metrics:  metrics,
	}
}

// Execute resolves the authenticated owner's store before retrieving the
// product. Missing stores, missing products, and ownership mismatches share
// ErrProductNotFound. Dependency errors are propagated. Store activity is
// deliberately not checked because owners can access products in inactive
// stores.
func (uc *GetMyProductByID) Execute(
	ctx context.Context,
	ownerID string,
	productID string,
) (*entities.Product, error) {
	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, ownerID)
	if err != nil {
		uc.record(ctx, "product.get_my_product_by_id.store_lookup_failed", "store")
		return nil, fmt.Errorf("get store for product owner: %w", err)
	}
	if storeID == "" {
		uc.record(ctx, "product.get_my_product_by_id.not_found", "")
		return nil, ErrProductNotFound
	}

	product, err := uc.products.GetByID(ctx, productID)
	if err != nil {
		uc.record(ctx, "product.get_my_product_by_id.repository_failed", "persistence")
		return nil, fmt.Errorf("get vendor product by ID: %w", err)
	}
	if product == nil {
		uc.record(ctx, "product.get_my_product_by_id.not_found", "")
		return nil, ErrProductNotFound
	}
	if product.StoreID() != storeID {
		uc.record(ctx, "product.get_my_product_by_id.ownership_mismatch", "")
		return nil, ErrProductNotFound
	}

	uc.record(ctx, "product.get_my_product_by_id.succeeded", "")
	return product, nil
}

// record keeps logging and metrics best-effort. Labels exclude owner, store,
// and product identifiers to avoid high-cardinality or sensitive values.
func (uc *GetMyProductByID) record(
	ctx context.Context,
	event string,
	failureCategory string,
) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event:           event,
			Operation:       "get_my_product_by_id",
			FailureCategory: failureCategory,
		})
	}

	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "get_my_product_by_id",
				"event":     event,
			},
		})
	}
}
