// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
)

// CountProductsByStore counts persisted products rather than relying on Store's
// cached capacity counter.
type CountProductsByStore struct {
	products ports.ProductRepository
	logger   ports.Logger
	metrics  ports.Metrics
}

func NewCountProductsByStore(
	products ports.ProductRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *CountProductsByStore {
	return &CountProductsByStore{products: products, logger: logger, metrics: metrics}
}

func (uc *CountProductsByStore) Execute(ctx context.Context, storeID string) (int64, error) {
	if !validIdentifier(storeID) {
		uc.record(ctx, "product.count_by_store.validation_failed", "validation")
		return 0, fmt.Errorf("store ID is required")
	}

	count, err := uc.products.CountByStore(ctx, storeID)
	if err != nil {
		uc.record(ctx, "product.count_by_store.repository_failed", "persistence")
		return 0, fmt.Errorf("count persisted store products: %w", err)
	}

	uc.record(ctx, "product.count_by_store.succeeded", "")
	return count, nil
}

func (uc *CountProductsByStore) record(ctx context.Context, event, failure string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "count_products_by_store", FailureCategory: failure,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name: "product_operations_total", Value: 1,
			Labels: map[string]string{"operation": "count_products_by_store", "event": event},
		})
	}
}
