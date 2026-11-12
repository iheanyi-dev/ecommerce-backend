// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
)

// ListProductsByStore lists publicly visible products for an active store.
type ListProductsByStore struct {
	products   ports.ProductRepository
	promotions ports.ProductPromotionRepository
	store      ports.StoreClient
	logger     ports.Logger
	metrics    ports.Metrics
}

func NewListProductsByStore(
	products ports.ProductRepository,
	store ports.StoreClient,
	logger ports.Logger,
	metrics ports.Metrics,
	promotions ports.ProductPromotionRepository,
) *ListProductsByStore {
	return &ListProductsByStore{
		products: products, promotions: promotions,
		store: store, logger: logger, metrics: metrics,
	}
}

func (uc *ListProductsByStore) Execute(
	ctx context.Context,
	storeID string,
	limit, offset int,
) (ProductPage, error) {
	limit, offset, err := normalizePagination(limit, offset)
	if err != nil || !validIdentifier(storeID) {
		uc.record(ctx, "product.list_by_store.validation_failed", "validation")
		if err != nil {
			return ProductPage{}, err
		}
		return ProductPage{}, fmt.Errorf("store ID is required")
	}

	active, err := uc.store.IsStoreActive(ctx, storeID)
	if err != nil {
		uc.record(ctx, "product.list_by_store.store_check_failed", "dependency")
		return ProductPage{}, fmt.Errorf("check store activity: %w", err)
	}
	if !active {
		uc.record(ctx, "product.list_by_store.store_inactive", "")
		return emptyProductPage(limit, offset), nil
	}

	products, total, err := uc.products.ListByStore(ctx, storeID, limit, offset)
	if err != nil {
		uc.record(ctx, "product.list_by_store.repository_failed", "persistence")
		return ProductPage{}, fmt.Errorf("list public store products: %w", err)
	}

	uc.record(ctx, "product.list_by_store.succeeded", "")
	results, err := loadProductResults(ctx, uc.promotions, products)
	if err != nil {
		uc.record(ctx, "product.list_by_store.promotion_lookup_failed", "dependency")
		return ProductPage{}, err
	}

	return ProductPage{Products: results, Total: total, Limit: limit, Offset: offset}, nil
}

func (uc *ListProductsByStore) record(ctx context.Context, event, failure string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "list_products_by_store", FailureCategory: failure,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name: "product_operations_total", Value: 1,
			Labels: map[string]string{"operation": "list_products_by_store", "event": event},
		})
	}
}
