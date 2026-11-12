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
)

type RemoveProductVideoInput struct {
	OwnerID   string
	ProductID string
}

type RemoveProductVideo struct {
	products ports.ProductRepository
	store    ports.StoreClient
	storage  ports.StorageClient
	logger   ports.Logger
	metrics  ports.Metrics
}

func NewRemoveProductVideo(
	products ports.ProductRepository,
	store ports.StoreClient,
	storage ports.StorageClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *RemoveProductVideo {
	return &RemoveProductVideo{
		products: products, store: store, storage: storage,
		logger: logger, metrics: metrics,
	}
}

func (uc *RemoveProductVideo) Execute(ctx context.Context, input RemoveProductVideoInput) error {
	ownerID := strings.TrimSpace(input.OwnerID)
	productID := strings.TrimSpace(input.ProductID)
	if ownerID == "" || productID == "" {
		uc.record(ctx, "product.remove_video.validation_failed", "validation")
		return errors.New("owner and product are required to remove product video")
	}

	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, ownerID)
	if err != nil {
		uc.record(ctx, "product.remove_video.owner_lookup_failed", "store")
		return fmt.Errorf("resolve product owner store: %w", err)
	}
	if strings.TrimSpace(storeID) == "" {
		uc.record(ctx, "product.remove_video.not_found", "")
		return ErrProductNotFound
	}

	product, err := uc.products.GetByID(ctx, productID)
	if err != nil {
		uc.record(ctx, "product.remove_video.repository_failed", "persistence")
		return fmt.Errorf("get product for video removal: %w", err)
	}
	if product == nil || product.StoreID() != storeID {
		uc.record(ctx, "product.remove_video.not_found", "")
		return ErrProductNotFound
	}

	active, err := uc.store.IsStoreActive(ctx, storeID)
	if err != nil {
		uc.record(ctx, "product.remove_video.store_check_failed", "store")
		return fmt.Errorf("check store status for video removal: %w", err)
	}
	if !active {
		uc.record(ctx, "product.remove_video.store_inactive", "")
		return ErrProductStoreInactive
	}

	previousVideo := product.Video()
	if previousVideo == nil {
		uc.record(ctx, "product.remove_video.not_found", "")
		return domainerrors.ErrVideoNotFound
	}
	if err := product.RemoveVideo(); err != nil {
		return err
	}
	if err := uc.products.Update(ctx, product); err != nil {
		_ = product.SetVideo(previousVideo.StorageReference)
		uc.record(ctx, "product.remove_video.persistence_failed", "persistence")
		return fmt.Errorf("persist product video removal: %w", err)
	}

	if err := uc.storage.DeleteObject(ctx, previousVideo.StorageReference); err != nil {
		uc.record(ctx, "product.remove_video.storage_cleanup_failed", "storage")
	}
	uc.record(ctx, "product.remove_video.succeeded", "")
	return nil
}

func (uc *RemoveProductVideo) record(ctx context.Context, event, failureCategory string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "remove_product_video", FailureCategory: failureCategory,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:   "product_operations_total",
			Value:  1,
			Labels: map[string]string{"operation": "remove_product_video", "event": event},
		})
	}
}

// Compile-time assertion that the use case operates on the Product aggregate.
var _ *entities.Product
