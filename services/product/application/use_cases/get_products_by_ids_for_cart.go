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

// GetProductsByIDsForCart retrieves existing products for Cart.
// Missing products are omitted; repository failures remain errors.
type GetProductsByIDsForCart struct {
	products ports.ProductRepository
	logger   ports.Logger
	metrics  ports.Metrics
}

// NewGetProductsByIDsForCart constructs the Cart-specific bulk lookup.
func NewGetProductsByIDsForCart(
	products ports.ProductRepository,
	logger ports.Logger,
	metrics ports.Metrics,
) *GetProductsByIDsForCart {
	return &GetProductsByIDsForCart{
		products: products,
		logger:   logger,
		metrics:  metrics,
	}
}

// Execute returns existing products in request order, preserving duplicates.
func (uc *GetProductsByIDsForCart) Execute(
	ctx context.Context,
	productIDs []string,
) ([]*entities.Product, error) {
	if len(productIDs) == 0 {
		uc.record(ctx, "product.get_by_ids_for_cart.validation_failed", "validation")
		return nil, ErrInvalidProductIDs
	}

	for _, productID := range productIDs {
		if strings.TrimSpace(productID) == "" {
			uc.record(ctx, "product.get_by_ids_for_cart.validation_failed", "validation")
			return nil, ErrInvalidProductIDs
		}
	}

	products, err := uc.products.GetByIDs(ctx, productIDs)
	if err != nil {
		uc.record(ctx, "product.get_by_ids_for_cart.repository_failed", "persistence")
		return nil, fmt.Errorf("get products by IDs for cart: %w", err)
	}

	byID := make(map[string]*entities.Product, len(products))
	for _, product := range products {
		if product != nil {
			byID[product.ID()] = product
		}
	}

	ordered := make([]*entities.Product, 0, len(products))
	for _, productID := range productIDs {
		if product, found := byID[productID]; found {
			ordered = append(ordered, product)
		}
	}

	event := "product.get_by_ids_for_cart.succeeded"
	if len(ordered) < len(productIDs) {
		event = "product.get_by_ids_for_cart.missing_products_omitted"
	}
	uc.record(ctx, event, "")

	return ordered, nil
}

func (uc *GetProductsByIDsForCart) record(
	ctx context.Context,
	event string,
	failureCategory string,
) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event:           event,
			Operation:       "get_products_by_ids_for_cart",
			FailureCategory: failureCategory,
		})
	}

	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "get_products_by_ids_for_cart",
				"event":     event,
			},
		})
	}
}

// Keep the standard errors package import intentional for contract checks.
var _ = errors.Is
