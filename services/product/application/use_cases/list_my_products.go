// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
)

// ListMyProducts lists products owned by the authenticated vendor, regardless
// of whether the vendor's store is currently active.
type ListMyProducts struct {
	products ports.ProductRepository
	store    ports.StoreClient
	logger   ports.Logger
	metrics  ports.Metrics
}

func NewListMyProducts(
	products ports.ProductRepository,
	store ports.StoreClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *ListMyProducts {
	return &ListMyProducts{
		products: products, store: store, logger: logger, metrics: metrics,
	}
}

func (uc *ListMyProducts) Execute(
	ctx context.Context,
	ownerID string,
	limit, offset int,
) (ProductPage, error) {
	limit, offset, err := normalizePagination(limit, offset)
	if err != nil || !validIdentifier(ownerID) {
		uc.record(ctx, "product.list_my_products.validation_failed", "validation")
		if err != nil {
			return ProductPage{}, err
		}
		return ProductPage{}, fmt.Errorf("owner ID is required")
	}

	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, ownerID)
	if err != nil {
		uc.record(ctx, "product.list_my_products.store_lookup_failed", "dependency")
		return ProductPage{}, fmt.Errorf("get vendor store: %w", err)
	}
	if storeID == "" {
		uc.record(ctx, "product.list_my_products.no_store", "")
		return emptyProductPage(limit, offset), nil
	}

	products, total, err := uc.products.ListByStore(ctx, storeID, limit, offset)
	if err != nil {
		uc.record(ctx, "product.list_my_products.repository_failed", "persistence")
		return ProductPage{}, fmt.Errorf("list vendor products: %w", err)
	}

	uc.record(ctx, "product.list_my_products.succeeded", "")
	return ProductPage{Products: products, Total: total, Limit: limit, Offset: offset}, nil
}

func (uc *ListMyProducts) record(ctx context.Context, event, failure string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "list_my_products", FailureCategory: failure,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name: "product_operations_total", Value: 1,
			Labels: map[string]string{"operation": "list_my_products", "event": event},
		})
	}
}
