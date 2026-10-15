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
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/domain/errors"
)

var ErrInvalidProductImage = errors.New("invalid product image")

// ChangeProductImageInput contains a new image and optionally the reference
// of an existing image to replace. An empty ExistingStorageReference adds one.
type ChangeProductImageInput struct {
	OwnerID                  string
	ProductID                string
	ExistingStorageReference string
	ContentType              string
	Content                  io.Reader
}

// ChangeProductImage uploads and attaches a product image, or replaces an
// existing image without changing its display position.
type ChangeProductImage struct {
	products ports.ProductRepository
	store    ports.StoreClient
	storage  ports.StorageClient
	logger   ports.Logger
	metrics  ports.Metrics
}

// NewChangeProductImage constructs the image use case.
func NewChangeProductImage(
	products ports.ProductRepository,
	store ports.StoreClient,
	storage ports.StorageClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *ChangeProductImage {
	return &ChangeProductImage{
		products: products,
		store:    store,
		storage:  storage,
		logger:   logger,
		metrics:  metrics,
	}
}

// Execute verifies ownership and store status, uploads the new object, then
// persists its reference. The old object is deleted only after persistence.
func (uc *ChangeProductImage) Execute(
	ctx context.Context,
	input ChangeProductImageInput,
) (*entities.Product, error) {
	input.OwnerID = strings.TrimSpace(input.OwnerID)
	input.ProductID = strings.TrimSpace(input.ProductID)
	input.ExistingStorageReference = strings.TrimSpace(input.ExistingStorageReference)

	extension, err := imageExtension(input.ContentType)
	if err != nil {
		uc.record(ctx, "product.change_image.validation_failed", "validation")
		return nil, err
	}
	if input.OwnerID == "" || input.ProductID == "" || input.Content == nil {
		uc.record(ctx, "product.change_image.validation_failed", "validation")
		return nil, fmt.Errorf("%w: owner, product, and image content are required", ErrInvalidProductImage)
	}

	storeID, err := uc.store.GetStoreIDByOwnerID(ctx, input.OwnerID)
	if err != nil {
		uc.record(ctx, "product.change_image.owner_lookup_failed", "store")
		return nil, fmt.Errorf("resolve product owner store: %w", err)
	}
	if strings.TrimSpace(storeID) == "" {
		uc.record(ctx, "product.change_image.not_found", "")
		return nil, ErrProductNotFound
	}

	product, err := uc.products.GetByID(ctx, input.ProductID)
	if err != nil {
		uc.record(ctx, "product.change_image.repository_failed", "persistence")
		return nil, fmt.Errorf("get product for image change: %w", err)
	}
	if product == nil || product.StoreID() != storeID {
		uc.record(ctx, "product.change_image.not_found", "")
		return nil, ErrProductNotFound
	}

	active, err := uc.store.IsStoreActive(ctx, storeID)
	if err != nil {
		uc.record(ctx, "product.change_image.store_check_failed", "store")
		return nil, fmt.Errorf("check store status for image change: %w", err)
	}
	if !active {
		uc.record(ctx, "product.change_image.store_inactive", "")
		return nil, errors.New("cannot change product images while store is inactive")
	}

	// Validate aggregate constraints before creating an object in Storage.
	if input.ExistingStorageReference == "" {
		if err := validateCanAddImage(product); err != nil {
			uc.record(ctx, "product.change_image.validation_failed", "validation")
			return nil, err
		}
	} else {
		if !containsProductImage(product, input.ExistingStorageReference) {
			uc.record(ctx, "product.change_image.image_not_found", "")
			return nil, domainerrors.ErrImageNotFound
		}
	}

	reference, err := newProductImageReference(product.ID(), extension)
	if err != nil {
		uc.record(ctx, "product.change_image.reference_generation_failed", "internal")
		return nil, fmt.Errorf("generate product image reference: %w", err)
	}

	if err := uc.storage.PutObject(ctx, reference, input.ContentType, input.Content); err != nil {
		uc.record(ctx, "product.change_image.upload_failed", "storage")
		return nil, fmt.Errorf("upload product image: %w", err)
	}

	// Product has no clone operation. Preserve its image list so we can restore
	// the aggregate if persistence fails.
	previousImages := product.Images()
	if input.ExistingStorageReference == "" {
		err = product.AddImage(reference)
	} else {
		err = product.ReplaceImage(input.ExistingStorageReference, reference)
	}
	if err != nil {
		_ = uc.storage.DeleteObject(ctx, reference)
		uc.record(ctx, "product.change_image.validation_failed", "validation")
		return nil, fmt.Errorf("attach uploaded product image: %w", err)
	}

	if err := uc.products.Update(ctx, product); err != nil {
		// Best-effort cleanup prevents an uploaded object from being orphaned.
		_ = uc.storage.DeleteObject(ctx, reference)

		// Restore the aggregate's original image state. Image order is preserved.
		restoreProductImages(product, previousImages)
		uc.record(ctx, "product.change_image.persistence_failed", "persistence")
		return nil, fmt.Errorf("persist product image change: %w", err)
	}

	if input.ExistingStorageReference != "" {
		if err := uc.storage.DeleteObject(ctx, input.ExistingStorageReference); err != nil {
			uc.record(ctx, "product.change_image.old_object_delete_failed", "storage")
		}
	}

	uc.record(ctx, "product.change_image.succeeded", "")
	return product, nil
}

func imageExtension(contentType string) (string, error) {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	switch contentType {
	case "image/jpeg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/webp":
		return ".webp", nil
	default:
		return "", fmt.Errorf("%w: unsupported content type %q", ErrInvalidProductImage, contentType)
	}
}

func newProductImageReference(productID, extension string) (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	imageID := fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		id[0:4], id[4:6], id[6:8], id[8:10], id[10:16],
	)
	return fmt.Sprintf("products/%s/%s%s", productID, imageID, extension), nil
}

func validateCanAddImage(product *entities.Product) error {
	if len(product.Images()) >= entities.MaxProductImages {
		return domainerrors.ErrImageLimitExceeded
	}
	return nil
}

func containsProductImage(product *entities.Product, reference string) bool {
	for _, image := range product.Images() {
		if image.StorageReference == reference {
			return true
		}
	}
	return false
}

// restoreProductImages reverses an add or replacement after failed persistence.
// It uses only the public aggregate methods, retaining the original ordering.
func restoreProductImages(product *entities.Product, images []entities.ProductImage) {
	current := product.Images()
	for _, image := range current {
		_ = product.RemoveImage(image.StorageReference)
	}
	for _, image := range images {
		_ = product.AddImage(image.StorageReference)
	}
}

func (uc *ChangeProductImage) record(ctx context.Context, event, failureCategory string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event:           event,
			Operation:       "change_product_image",
			FailureCategory: failureCategory,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "change_product_image",
				"event":     event,
			},
		})
	}
}
