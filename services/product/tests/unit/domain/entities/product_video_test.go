package entities_test

import (
	"errors"
	"testing"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/domain/errors"
)

func TestProductSetsVideo(t *testing.T) {
	product := newValidProduct(t, 5)

	if product.Video() != nil {
		t.Fatal("new product should have no video")
	}

	if err := product.SetVideo(" products/product-123/video.mp4 "); err != nil {
		t.Fatal(err)
	}

	video := product.Video()
	if video == nil || video.StorageReference != "products/product-123/video.mp4" {
		t.Fatalf("unexpected video reference: %#v", video)
	}
}

func TestProductReplacesVideo(t *testing.T) {
	product := newValidProduct(t, 5)

	if err := product.SetVideo("products/product-123/old.mp4"); err != nil {
		t.Fatal(err)
	}
	if err := product.SetVideo("products/product-123/new.mp4"); err != nil {
		t.Fatal(err)
	}

	if got := product.Video().StorageReference; got != "products/product-123/new.mp4" {
		t.Fatalf("video reference = %q", got)
	}
}

func TestProductRejectsBlankVideoReference(t *testing.T) {
	product := newValidProduct(t, 5)

	err := product.SetVideo(" ")
	if !errors.Is(err, domainerrors.ErrInvalidVideo) {
		t.Fatalf("error = %v, want ErrInvalidVideo", err)
	}
}

func TestProductRemovesVideo(t *testing.T) {
	product := newValidProduct(t, 5)

	if err := product.SetVideo("products/product-123/video.mp4"); err != nil {
		t.Fatal(err)
	}
	if err := product.RemoveVideo(); err != nil {
		t.Fatal(err)
	}
	if product.Video() != nil {
		t.Fatal("video should be nil after removal")
	}
}

func TestProductRemoveVideoWhenAbsent(t *testing.T) {
	product := newValidProduct(t, 5)

	err := product.RemoveVideo()
	if !errors.Is(err, domainerrors.ErrVideoNotFound) {
		t.Fatalf("error = %v, want ErrVideoNotFound", err)
	}
}

func TestProductVideoReturnsCopy(t *testing.T) {
	product := newValidProduct(t, 5)

	if err := product.SetVideo("products/product-123/video.mp4"); err != nil {
		t.Fatal(err)
	}

	video := product.Video()
	video.StorageReference = "tampered"

	if got := product.Video().StorageReference; got != "products/product-123/video.mp4" {
		t.Fatalf("aggregate video reference was mutated: %q", got)
	}
}
