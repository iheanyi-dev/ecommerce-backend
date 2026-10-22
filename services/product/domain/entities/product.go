// Package entities contains the Product aggregate and its domain behavior.
package entities

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

const MaxProductImages = 4

// ProductImage is a reference to an image managed by the Storage Service.
// Image bytes and storage operations do not belong in the Product domain.
type ProductImage struct {
	StorageReference string
}

// Product is the aggregate root for a marketplace product.
//
// Its image slice order determines image display order. The first image,
// when present, is the primary image.
type Product struct {
	id          string
	storeID     string
	name        string
	description string
	price       valueobjects.Money
	quantity    int64
	images      []ProductImage
	createdAt   time.Time
	updatedAt   time.Time
}

// NewProduct creates a product that is immediately eligible for discovery.
//
// A product starts with no images. Vendors can add, remove, or reorder images
// independently after creation. Zero quantity is valid and does not hide it.
func NewProduct(
	storeID string,
	name string,
	description string,
	price valueobjects.Money,
	quantity int64,
) (*Product, error) {
	storeID = strings.TrimSpace(storeID)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if storeID == "" {
		return nil, fmt.Errorf("%w: store ID is required", domainerrors.ErrInvalidProduct)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", domainerrors.ErrInvalidProduct)
	}
	if description == "" {
		return nil, fmt.Errorf("%w: description is required", domainerrors.ErrInvalidProduct)
	}
	if quantity < 0 {
		return nil, fmt.Errorf("%w: quantity cannot be negative", domainerrors.ErrInvalidProduct)
	}

	id, err := newProductID()
	if err != nil {
		return nil, fmt.Errorf("generate product ID: %w", err)
	}

	now := time.Now().UTC()

	return &Product{
		id:          id,
		storeID:     storeID,
		name:        name,
		description: description,
		price:       price,
		quantity:    quantity,
		images:      make([]ProductImage, 0, MaxProductImages),
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ReconstituteProduct restores a Product aggregate from persisted state.
// It preserves the persisted identity and timestamps without generating a new ID.
func ReconstituteProduct(
	id, storeID, name, description string,
	price valueobjects.Money,
	quantity int64,
	images []ProductImage,
	createdAt, updatedAt time.Time,
) (*Product, error) {
	id = strings.TrimSpace(id)
	storeID = strings.TrimSpace(storeID)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if id == "" || storeID == "" || name == "" || description == "" || quantity < 0 {
		return nil, fmt.Errorf("%w: invalid persisted product fields", domainerrors.ErrInvalidProduct)
	}
	if _, err := valueobjects.NewMoney(price.Amount(), price.Currency()); err != nil {
		return nil, fmt.Errorf("%w: invalid persisted price: %v", domainerrors.ErrInvalidProduct, err)
	}
	if createdAt.IsZero() || updatedAt.IsZero() || updatedAt.Before(createdAt) {
		return nil, fmt.Errorf("%w: invalid persisted timestamps", domainerrors.ErrInvalidProduct)
	}
	if len(images) > MaxProductImages {
		return nil, domainerrors.ErrImageLimitExceeded
	}

	restoredImages := make([]ProductImage, 0, MaxProductImages)
	seen := make(map[string]struct{}, len(images))
	for _, image := range images {
		reference := strings.TrimSpace(image.StorageReference)
		if reference == "" {
			return nil, domainerrors.ErrInvalidImage
		}
		if _, exists := seen[reference]; exists {
			return nil, domainerrors.ErrDuplicateImage
		}
		seen[reference] = struct{}{}
		restoredImages = append(restoredImages, ProductImage{StorageReference: reference})
	}

	return &Product{
		id: id, storeID: storeID, name: name, description: description,
		price: price, quantity: quantity, images: restoredImages,
		createdAt: createdAt.UTC(), updatedAt: updatedAt.UTC(),
	}, nil
}

// newProductID generates a UUID v4 using the standard library.
func newProductID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}

	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		id[0:4], id[4:6], id[6:8], id[8:10], id[10:16],
	), nil
}

// ID returns the system-generated product identifier.
func (p *Product) ID() string { return p.id }

// StoreID returns the owning store identifier.
func (p *Product) StoreID() string { return p.storeID }

// Name returns the product name.
func (p *Product) Name() string { return p.name }

// Description returns the product description.
func (p *Product) Description() string { return p.description }

// Price returns the product price by value.
func (p *Product) Price() valueobjects.Money { return p.price }

// Quantity returns the amount available for ordering.
func (p *Product) Quantity() int64 { return p.quantity }

// Images returns a copy so callers cannot mutate aggregate state.
func (p *Product) Images() []ProductImage {
	images := make([]ProductImage, len(p.images))
	copy(images, p.images)
	return images
}

// CreatedAt returns the product creation time.
func (p *Product) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns the last successful modification time.
func (p *Product) UpdatedAt() time.Time { return p.updatedAt }

// UpdateDetails changes the product's name and description.
func (p *Product) UpdateDetails(name, description string) error {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return fmt.Errorf("%w: name is required", domainerrors.ErrInvalidProduct)
	}
	if description == "" {
		return fmt.Errorf("%w: description is required", domainerrors.ErrInvalidProduct)
	}

	if p.name == name && p.description == description {
		return nil
	}

	p.name = name
	p.description = description
	p.touch()
	return nil
}

// UpdatePrice changes the product price.
func (p *Product) UpdatePrice(price valueobjects.Money) {
	if p.price.Equal(price) {
		return
	}

	p.price = price
	p.touch()
}

// UpdateQuantity changes the quantity available for ordering.
func (p *Product) UpdateQuantity(quantity int64) error {
	if quantity < 0 {
		return fmt.Errorf("%w: quantity cannot be negative", domainerrors.ErrInvalidProduct)
	}
	if p.quantity == quantity {
		return nil
	}

	p.quantity = quantity
	p.touch()
	return nil
}

// AddImage adds a Storage Service reference while enforcing the four-image cap.
func (p *Product) AddImage(reference string) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return domainerrors.ErrInvalidImage
	}
	if len(p.images) >= MaxProductImages {
		return domainerrors.ErrImageLimitExceeded
	}
	if p.hasImage(reference) {
		return domainerrors.ErrDuplicateImage
	}

	p.images = append(p.images, ProductImage{StorageReference: reference})
	p.touch()
	return nil
}

// ReplaceImage replaces an existing Storage reference without changing its
// display position. The aggregate's four-image limit remains unchanged.
func (p *Product) ReplaceImage(oldReference, newReference string) error {
	newReference = strings.TrimSpace(newReference)
	if newReference == "" {
		return domainerrors.ErrInvalidImage
	}

	oldIndex := -1
	for i, image := range p.images {
		if image.StorageReference == oldReference {
			oldIndex = i
			break
		}
	}
	if oldIndex == -1 {
		return domainerrors.ErrImageNotFound
	}

	if oldReference == newReference {
		return nil
	}
	if p.hasImage(newReference) {
		return domainerrors.ErrDuplicateImage
	}

	p.images[oldIndex] = ProductImage{StorageReference: newReference}
	p.touch()
	return nil
}

// RemoveImage removes an image by its Storage Service reference.
func (p *Product) RemoveImage(reference string) error {
	for i, image := range p.images {
		if image.StorageReference == reference {
			p.images = append(p.images[:i], p.images[i+1:]...)
			p.touch()
			return nil
		}
	}

	return domainerrors.ErrImageNotFound
}

// ReorderImages sets the complete display order of the existing images.
//
// The supplied references must contain every current image exactly once.
// Partial lists, unknown references, and duplicates are rejected.
func (p *Product) ReorderImages(references []string) error {
	if len(references) != len(p.images) {
		return domainerrors.ErrInvalidImageOrder
	}

	current := make(map[string]ProductImage, len(p.images))
	for _, image := range p.images {
		current[image.StorageReference] = image
	}

	reordered := make([]ProductImage, 0, len(references))
	seen := make(map[string]struct{}, len(references))

	for _, reference := range references {
		if _, duplicate := seen[reference]; duplicate {
			return domainerrors.ErrInvalidImageOrder
		}

		image, exists := current[reference]
		if !exists {
			return domainerrors.ErrInvalidImageOrder
		}

		seen[reference] = struct{}{}
		reordered = append(reordered, image)
	}

	changed := false
	for i := range p.images {
		if p.images[i].StorageReference != reordered[i].StorageReference {
			changed = true
			break
		}
	}

	if changed {
		p.images = reordered
		p.touch()
	}

	return nil
}

func (p *Product) hasImage(reference string) bool {
	for _, image := range p.images {
		if image.StorageReference == reference {
			return true
		}
	}
	return false
}

func (p *Product) touch() {
	p.updatedAt = time.Now().UTC()
}
