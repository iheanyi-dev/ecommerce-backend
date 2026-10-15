// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

var (
	// ErrInvalidProductUpdate indicates that an update has no fields or
	// contains invalid field values.
	ErrInvalidProductUpdate = errors.New("invalid product update")

	// ErrProductStoreInactive indicates that a product cannot be updated
	// while its owning store is inactive.
	ErrProductStoreInactive = errors.New("cannot update products while store is inactive")
)

// UpdateProductInput contains optional fields to update. Nil means unchanged;
// pointers allow callers to explicitly set quantity to zero.
type UpdateProductInput struct {
	OwnerID     string
	ProductID   string
	Name        *string
	Description *string
	Price       *valueobjects.Money
	Quantity    *int64
}

// UpdateProduct updates an owner's product and refreshes its embedding when
// its searchable text changes.
type UpdateProduct struct {
	products  ports.ProductRepository
	store     ports.StoreClient
	embedding ports.EmbeddingDispatcher
	logger    ports.Logger
	metrics   ports.Metrics
}

// NewUpdateProduct constructs the product update use case.
func NewUpdateProduct(
	products ports.ProductRepository,
	store ports.StoreClient,
	embedding ports.EmbeddingDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *UpdateProduct {
	return &UpdateProduct{
		products:  products,
		store:     store,
		embedding: embedding,
		logger:    logger,
		metrics:   metrics,
	}
}

// Execute validates ownership and store status, persists the requested
// changes, and best-effort dispatches an embedding update when text changes.
func (uc *UpdateProduct) Execute(
	ctx context.Context,
	input UpdateProductInput,
) (*entities.Product, error) {
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.ProductID = strings.TrimSpace(input.ProductID)

	if input.OwnerID == "" || input.ProductID == "" ||
		(input.Name == nil && input.Description == nil &&
			input.Price == nil && input.Quantity == nil) {
		uc.record(ctx, "product.update.invalid_input", "validation")
		return nil, ErrInvalidProductUpdate
	}
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		uc.record(ctx, "product.update.invalid_input", "validation")
		return nil, fmt.Errorf("%w: name cannot be empty", ErrInvalidProductUpdate)
	}
	if input.Description != nil && strings.TrimSpace(*input.Description) == "" {
		uc.record(ctx, "product.update.invalid_input", "validation")
		return nil, fmt.Errorf("%w: description cannot be empty", ErrInvalidProductUpdate)
	}
	if input.Quantity != nil && *input.Quantity < 0 {
		uc.record(ctx, "product.update.invalid_input", "validation")
		return nil, fmt.Errorf("%w: quantity cannot be negative", ErrInvalidProductUpdate)
	}

	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, input.OwnerID)
	if err != nil {
		uc.record(ctx, "product.update.owner_lookup_failed", "store")
		return nil, fmt.Errorf("resolve owner's store: %w", err)
	}
	if strings.TrimSpace(storeID) == "" {
		uc.record(ctx, "product.update.not_found", "")
		return nil, ErrProductNotFound
	}

	product, err := uc.products.GetByID(ctx, input.ProductID)
	if err != nil {
		uc.record(ctx, "product.update.repository_failed", "persistence")
		return nil, fmt.Errorf("get product for update: %w", err)
	}
	if product == nil || product.StoreID() != storeID {
		uc.record(ctx, "product.update.not_found", "")
		return nil, ErrProductNotFound
	}

	active, err := uc.store.IsStoreActive(ctx, storeID)
	if err != nil {
		uc.record(ctx, "product.update.store_check_failed", "store")
		return nil, fmt.Errorf("check store activity for product update: %w", err)
	}
	if !active {
		uc.record(ctx, "product.update.store_inactive", "")
		return nil, ErrProductStoreInactive
	}

	oldName, oldDescription := product.Name(), product.Description()

	name := oldName
	description := oldDescription
	if input.Name != nil {
		name = *input.Name
	}
	if input.Description != nil {
		description = *input.Description
	}
	if name != oldName || description != oldDescription {
		if err := product.UpdateDetails(name, description); err != nil {
			uc.record(ctx, "product.update.validation_failed", "validation")
			return nil, fmt.Errorf("update product details: %w", err)
		}
	}
	if input.Price != nil {
		product.UpdatePrice(*input.Price)
	}
	if input.Quantity != nil {
		if err := product.UpdateQuantity(*input.Quantity); err != nil {
			uc.record(ctx, "product.update.validation_failed", "validation")
			return nil, fmt.Errorf("update product quantity: %w", err)
		}
	}

	if err := uc.products.Update(ctx, product); err != nil {
		uc.record(ctx, "product.update.persistence_failed", "persistence")
		return nil, fmt.Errorf("persist product update: %w", err)
	}

	if (product.Name() != oldName || product.Description() != oldDescription) && uc.embedding != nil {
		if err := uc.embedding.DispatchProductEmbedding(ctx, ports.ProductEmbedding{
			ProductID:   product.ID(),
			StoreID:     product.StoreID(),
			Name:        product.Name(),
			Description: product.Description(),
		}); err != nil {
			uc.record(ctx, "product.update.embedding_dispatch_failed", "embedding")
		}
	}

	uc.record(ctx, "product.update.succeeded", "")
	return product, nil
}

// record emits best-effort logs and metrics without high-cardinality IDs.
func (uc *UpdateProduct) record(ctx context.Context, event, failureCategory string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "update_product", FailureCategory: failureCategory,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "update_product",
				"event":     event,
			},
		})
	}
}

// Keep domain validation errors discoverable to callers if future domain
// validation rules are introduced.
var _ = domainerrors.ErrInvalidProduct
