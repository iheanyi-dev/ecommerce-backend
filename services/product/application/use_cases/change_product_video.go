// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

var ErrInvalidProductVideo = errors.New("invalid product video")

type ChangeProductVideoInput struct {
	OwnerID     string
	ProductID   string
	ContentType string
	Content     io.Reader
}

type ChangeProductVideo struct {
	products ports.ProductRepository
	store    ports.StoreClient
	storage  ports.StorageClient
	logger   ports.Logger
	metrics  ports.Metrics
}

func NewChangeProductVideo(
	products ports.ProductRepository,
	store ports.StoreClient,
	storage ports.StorageClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *ChangeProductVideo {
	return &ChangeProductVideo{
		products: products,
		store:    store,
		storage:  storage,
		logger:   logger,
		metrics:  metrics,
	}
}

func (uc *ChangeProductVideo) Execute(
	ctx context.Context,
	input ChangeProductVideoInput,
) (*entities.Product, error) {
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.ProductID = strings.TrimSpace(input.ProductID)
	input.ContentType = strings.TrimSpace(strings.ToLower(input.ContentType))

	extension, err := productVideoExtension(input.ContentType)
	if err != nil {
		uc.record(ctx, "product.change_video.validation_failed", "validation")
		return nil, err
	}
	if input.OwnerID == "" || input.ProductID == "" || input.Content == nil {
		uc.record(ctx, "product.change_video.validation_failed", "validation")
		return nil, fmt.Errorf("%w: owner, product, and video content are required", ErrInvalidProductVideo)
	}

	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, input.OwnerID)
	if err != nil {
		uc.record(ctx, "product.change_video.owner_lookup_failed", "store")
		return nil, fmt.Errorf("resolve product owner store: %w", err)
	}
	if strings.TrimSpace(storeID) == "" {
		uc.record(ctx, "product.change_video.not_found", "")
		return nil, ErrProductNotFound
	}

	product, err := uc.products.GetByID(ctx, input.ProductID)
	if err != nil {
		uc.record(ctx, "product.change_video.repository_failed", "persistence")
		return nil, fmt.Errorf("get product for video change: %w", err)
	}
	if product == nil || product.StoreID() != storeID {
		uc.record(ctx, "product.change_video.not_found", "")
		return nil, ErrProductNotFound
	}

	active, err := uc.store.IsStoreActive(ctx, storeID)
	if err != nil {
		uc.record(ctx, "product.change_video.store_check_failed", "store")
		return nil, fmt.Errorf("check store status for video change: %w", err)
	}
	if !active {
		uc.record(ctx, "product.change_video.store_inactive", "")
		return nil, ErrProductStoreInactive
	}

	reference, err := newProductVideoReference(product.ID(), extension)
	if err != nil {
		uc.record(ctx, "product.change_video.reference_generation_failed", "internal")
		return nil, fmt.Errorf("generate product video reference: %w", err)
	}
	if err := uc.storage.PutObject(ctx, reference, input.ContentType, input.Content); err != nil {
		uc.record(ctx, "product.change_video.upload_failed", "storage")
		return nil, fmt.Errorf("upload product video: %w", err)
	}

	previousVideo := product.Video()
	if err := product.SetVideo(reference); err != nil {
		_ = uc.storage.DeleteObject(ctx, reference)
		uc.record(ctx, "product.change_video.validation_failed", "validation")
		return nil, fmt.Errorf("attach uploaded product video: %w", err)
	}

	if err := uc.products.Update(ctx, product); err != nil {
		_ = uc.storage.DeleteObject(ctx, reference)
		if previousVideo == nil {
			_ = product.RemoveVideo()
		} else {
			_ = product.SetVideo(previousVideo.StorageReference)
		}
		uc.record(ctx, "product.change_video.persistence_failed", "persistence")
		return nil, fmt.Errorf("persist product video change: %w", err)
	}

	if previousVideo != nil {
		if err := uc.storage.DeleteObject(ctx, previousVideo.StorageReference); err != nil {
			uc.record(ctx, "product.change_video.old_object_delete_failed", "storage")
		}
	}

	uc.record(ctx, "product.change_video.succeeded", "")
	return product, nil
}

func productVideoExtension(contentType string) (string, error) {
	switch contentType {
	case "video/mp4":
		return ".mp4", nil
	case "video/quicktime":
		return ".mov", nil
	case "video/webm":
		return ".webm", nil
	default:
		return "", fmt.Errorf("%w: unsupported content type %q", ErrInvalidProductVideo, contentType)
	}
}

func newProductVideoReference(productID, extension string) (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	videoID := fmt.Sprintf("%x-%x-%x-%x-%x",
		id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
	return fmt.Sprintf("products/%s/%s%s", productID, videoID, extension), nil
}

func (uc *ChangeProductVideo) record(ctx context.Context, event, failureCategory string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "change_product_video", FailureCategory: failureCategory,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:   "product_operations_total",
			Value:  1,
			Labels: map[string]string{"operation": "change_product_video", "event": event},
		})
	}
}
