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

// ErrInvalidProductDeletion indicates that required deletion input is absent.
var ErrInvalidProductDeletion = errors.New("invalid product deletion")

// DeleteProductInput identifies the owner and product to delete.
type DeleteProductInput struct {
	OwnerID   string
	ProductID string
}

// DeleteProduct removes a product owned by a user and cleans up its resources.
type DeleteProduct struct {
	products  ports.ProductRepository
	store     ports.StoreClient
	storage   ports.StorageClient
	embedding ports.EmbeddingDispatcher
	logger    ports.Logger
	metrics   ports.Metrics
}

// NewDeleteProduct constructs the product deletion use case.
func NewDeleteProduct(
	products ports.ProductRepository,
	store ports.StoreClient,
	storage ports.StorageClient,
	embedding ports.EmbeddingDispatcher,
	logger ports.Logger,
	metrics ports.Metrics,
) *DeleteProduct {
	return &DeleteProduct{
		products:  products,
		store:     store,
		storage:   storage,
		embedding: embedding,
		logger:    logger,
		metrics:   metrics,
	}
}

// Execute decrements store capacity before deleting the product. Resource
// cleanup happens only after repository deletion succeeds. Deletion is
// permitted even if the owning store is inactive.
func (uc *DeleteProduct) Execute(ctx context.Context, input DeleteProductInput) error {
	ownerID := strings.TrimSpace(input.OwnerID)
	productID := strings.TrimSpace(input.ProductID)
	if ownerID == "" || productID == "" {
		uc.record(ctx, "product.delete.invalid_input", "validation")
		return ErrInvalidProductDeletion
	}

	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, ownerID)
	if err != nil {
		uc.record(ctx, "product.delete.owner_lookup_failed", "store")
		return fmt.Errorf("resolve owner's store: %w", err)
	}
	if strings.TrimSpace(storeID) == "" {
		uc.record(ctx, "product.delete.not_found", "")
		return ErrProductNotFound
	}

	product, err := uc.products.GetByID(ctx, productID)
	if err != nil {
		uc.record(ctx, "product.delete.repository_failed", "persistence")
		return fmt.Errorf("get product for deletion: %w", err)
	}
	if product == nil || product.StoreID() != storeID {
		uc.record(ctx, "product.delete.not_found", "")
		return ErrProductNotFound
	}

	if err := uc.store.DecrementProductCount(ctx, storeID); err != nil {
		uc.record(ctx, "product.delete.capacity_decrement_failed", "store")
		return fmt.Errorf("decrement store product count: %w", err)
	}

	if err := uc.products.Delete(ctx, productID); err != nil {
		uc.record(ctx, "product.delete.persistence_failed", "persistence")
		return fmt.Errorf("delete product: %w", err)
	}

	// External cleanup is best-effort after the product has been deleted.
	if uc.embedding != nil {
		if err := uc.embedding.DispatchProductEmbeddingDeletion(ctx, product.ID(), product.StoreID()); err != nil {
			uc.record(ctx, "product.delete.embedding_cleanup_failed", "embedding")
		}
	}

	if uc.storage != nil {
		for _, image := range product.Images() {
			if err := uc.storage.DeleteObject(ctx, image.StorageReference); err != nil {
				uc.record(ctx, "product.delete.image_cleanup_failed", "storage")
			}
		}
		if video := product.Video(); video != nil {
			if err := uc.storage.DeleteObject(ctx, video.StorageReference); err != nil {
				uc.record(ctx, "product.delete.video_cleanup_failed", "storage")
			}
		}
	}

	uc.record(ctx, "product.delete.succeeded", "")
	return nil
}

// record emits best-effort logs and metrics without high-cardinality IDs.
func (uc *DeleteProduct) record(ctx context.Context, event, failureCategory string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "delete_product", FailureCategory: failureCategory,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "delete_product",
				"event":     event,
			},
		})
	}
}

// Compile-time assertion that the product aggregate remains the cleanup source.
var _ *entities.Product
