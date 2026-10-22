// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

// ErrProductCreationOutcomeUnknown indicates that persistence may have
// succeeded even though the repository returned an error. In this situation,
// Store capacity must not be decremented blindly.
var (
	// ErrProductCreationOutcomeUnknown indicates that persistence may have
	// succeeded even though the repository returned an error.
	ErrProductCreationOutcomeUnknown = errors.New("product creation outcome unknown")

	// ErrProductOwnerStoreNotFound indicates that the authenticated owner
	// does not have a store in which to create a product.
	ErrProductOwnerStoreNotFound = errors.New("owner store not found")

	// ErrInvalidProductOwner indicates that no authenticated owner was supplied.
	ErrInvalidProductOwner = errors.New("product owner ID is required")
)

// CreateProductInput contains the data required to create a product.
// Images are uploaded and attached in a separate workflow.
type CreateProductInput struct {
	OwnerID     string
	Name        string
	Description string
	Price       valueobjects.Money
	Quantity    int64
}

// CreateProduct creates and persists a product after reserving Store capacity.
type CreateProduct struct {
	products   ports.ProductRepository
	store      ports.StoreClient
	embeddings ports.EmbeddingDispatcher
	logger     ports.Logger
	metrics    ports.Metrics
}

// NewCreateProduct constructs the CreateProduct use case.
func NewCreateProduct(
	products ports.ProductRepository,
	store ports.StoreClient,
	embeddings ports.EmbeddingDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *CreateProduct {
	return &CreateProduct{
		products:   products,
		store:      store,
		embeddings: embeddings,
		logger:     logger,
		metrics:    metrics,
	}
}

// Execute validates the input, reserves Store capacity, and persists the
// product. Embedding dispatch is best-effort after successful persistence.
func (uc *CreateProduct) Execute(
	ctx context.Context,
	input CreateProductInput,
) (*entities.Product, error) {
	if input.OwnerID == "" {
		uc.record(ctx, "product.create.validation_failed", "create_product", "validation")
		return nil, ErrInvalidProductOwner
	}

	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, input.OwnerID)
	if err != nil {
		uc.record(ctx, "product.create.store_lookup_failed", "create_product", "store")
		return nil, fmt.Errorf("resolve owner store: %w", err)
	}
	if storeID == "" {
		uc.record(ctx, "product.create.store_not_found", "create_product", "validation")
		return nil, ErrProductOwnerStoreNotFound
	}

	product, err := entities.NewProduct(
		storeID,
		input.Name,
		input.Description,
		input.Price,
		input.Quantity,
	)
	if err != nil {
		uc.record(ctx, "product.create.validation_failed", "create_product", "validation")
		return nil, fmt.Errorf("create product entity: %w", err)
	}

	if err := uc.store.IncrementProductCount(ctx, product.StoreID()); err != nil {
		uc.record(ctx, "product.create.capacity_reservation_failed", "create_product", "store")
		return nil, fmt.Errorf("reserve store product capacity: %w", err)
	}

	if err := uc.products.Create(ctx, product); err != nil {
		// An ambiguous persistence result must not trigger compensation:
		// the product may already exist despite the returned error.
		if errors.Is(err, ErrProductCreationOutcomeUnknown) {
			uc.record(ctx, "product.create.outcome_unknown", "create_product", "persistence")
			return nil, fmt.Errorf("persist product; outcome unknown: %w", err)
		}

		if compensationErr := uc.store.DecrementProductCount(ctx, product.StoreID()); compensationErr != nil {
			uc.record(ctx, "product.create.capacity_compensation_failed", "create_product", "compensation")
			return nil, fmt.Errorf(
				"persist product: %w; release reserved store capacity: %v",
				err,
				compensationErr,
			)
		}

		uc.record(ctx, "product.create.persistence_failed", "create_product", "persistence")
		return nil, fmt.Errorf("persist product: %w", err)
	}

	if err := uc.embeddings.DispatchProductEmbedding(ctx, ports.ProductEmbedding{
		ProductID:   product.ID(),
		StoreID:     product.StoreID(),
		Name:        product.Name(),
		Description: product.Description(),
	}); err != nil {
		// Product creation has succeeded. Embedding failure must not make the
		// caller believe the product was not created.
		uc.record(ctx, "product.create.embedding_dispatch_failed", "create_product", "embedding")
	}

	uc.record(ctx, "product.create.succeeded", "create_product", "")
	return product, nil
}

// record makes observability best-effort so logging and metrics outages
// cannot alter the business operation's result.
func (uc *CreateProduct) record(
	ctx context.Context,
	event string,
	operation string,
	failureCategory string,
) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event:           event,
			Operation:       operation,
			FailureCategory: failureCategory,
		})
	}

	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": operation,
				"event":     event,
			},
		})
	}
}
