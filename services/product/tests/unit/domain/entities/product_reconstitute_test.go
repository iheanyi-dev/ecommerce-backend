package entities_test

import (
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

func TestReconstituteProductPreservesPersistedState(t *testing.T) {
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := created.Add(time.Hour)
	price, err := valueobjects.NewMoney(1250, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	images := []entities.ProductImage{
		{StorageReference: "products/primary.webp"},
		{StorageReference: "products/secondary.webp"},
	}
	video := &entities.ProductVideo{
		StorageReference: "products/short-video.mp4",
	}

	product, err := entities.ReconstituteProduct(
		"persisted-id", "persisted-store", " Product ", " Description ",
		price, 7, images, video, created, updated,
	)
	if err != nil {
		t.Fatal(err)
	}
	if product.ID() != "persisted-id" || product.StoreID() != "persisted-store" {
		t.Fatal("persisted identifiers were not preserved")
	}
	if product.Name() != "Product" || product.Description() != "Description" {
		t.Fatal("text fields were not normalized")
	}
	if !product.CreatedAt().Equal(created) || !product.UpdatedAt().Equal(updated) {
		t.Fatal("persisted timestamps were not preserved")
	}
	got := product.Images()
	if len(got) != 2 || got[0] != images[0] || got[1] != images[1] {
		t.Fatalf("image order was not preserved: %#v", got)
	}
	images[0].StorageReference = "mutated"
	if product.Images()[0].StorageReference != "products/primary.webp" {
		t.Fatal("aggregate aliases caller image memory")
	}

	gotVideo := product.Video()
	if gotVideo == nil || gotVideo.StorageReference != video.StorageReference {
		t.Fatalf("video reference was not preserved: %#v", gotVideo)
	}
	gotVideo.StorageReference = "mutated"
	if product.Video().StorageReference != video.StorageReference {
		t.Fatal("Video() exposes aggregate state")
	}
}

func TestReconstituteProductRejectsInvalidState(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	price, err := valueobjects.NewMoney(1250, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name             string
		id               string
		quantity         int64
		images           []entities.ProductImage
		video            *entities.ProductVideo
		created, updated time.Time
		want             error
	}{
		{"missing ID", "", 1, nil, nil, now, now, domainerrors.ErrInvalidProduct},
		{"negative quantity", "id", -1, nil, nil, now, now, domainerrors.ErrInvalidProduct},
		{"missing timestamps", "id", 1, nil, nil, time.Time{}, time.Time{}, domainerrors.ErrInvalidProduct},
		{"updated before created", "id", 1, nil, nil, now, now.Add(-time.Second), domainerrors.ErrInvalidProduct},
		{"blank image", "id", 1, []entities.ProductImage{{StorageReference: " "}}, nil, now, now, domainerrors.ErrInvalidImage},
		{"duplicate images", "id", 1, []entities.ProductImage{{StorageReference: "same"}, {StorageReference: "same"}}, nil, now, now, domainerrors.ErrDuplicateImage},
		{"too many images", "id", 1, []entities.ProductImage{{StorageReference: "1"}, {StorageReference: "2"}, {StorageReference: "3"}, {StorageReference: "4"}, {StorageReference: "5"}}, nil, now, now, domainerrors.ErrImageLimitExceeded},
		{"blank video", "id", 1, nil, &entities.ProductVideo{StorageReference: " "}, now, now, domainerrors.ErrInvalidVideo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.ReconstituteProduct(
				tt.id, "store-id", "name", "description", price,
				tt.quantity, tt.images, tt.video, tt.created, tt.updated,
			)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v; want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}
