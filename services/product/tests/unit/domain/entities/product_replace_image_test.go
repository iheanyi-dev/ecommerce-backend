package entities_test

import (
	"errors"
	"testing"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/domain/errors"
)

func TestReplaceImagePreservesDisplayPosition(t *testing.T) {
	product := newValidProduct(t, 5)
	refs := []string{
		"products/product-123/front.webp",
		"products/product-123/side.webp",
		"products/product-123/back.webp",
	}

	for _, ref := range refs {
		if err := product.AddImage(ref); err != nil {
			t.Fatal(err)
		}
	}

	replacement := "products/product-123/replacement.webp"
	if err := product.ReplaceImage(refs[1], replacement); err != nil {
		t.Fatalf("ReplaceImage(): %v", err)
	}

	got := product.Images()
	want := []string{refs[0], replacement, refs[2]}
	if len(got) != len(want) {
		t.Fatalf("image count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].StorageReference != want[i] {
			t.Errorf("image at position %d = %q, want %q", i, got[i].StorageReference, want[i])
		}
	}
}

func TestReplaceImageRejectsInvalidReferences(t *testing.T) {
	product := newValidProduct(t, 5)
	existing := "products/product-123/front.webp"
	other := "products/product-123/side.webp"

	if err := product.AddImage(existing); err != nil {
		t.Fatal(err)
	}
	if err := product.AddImage(other); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		oldRef  string
		newRef  string
		wantErr error
	}{
		{
			name:    "old reference does not exist",
			oldRef:  "products/product-123/missing.webp",
			newRef:  "products/product-123/new.webp",
			wantErr: domainerrors.ErrImageNotFound,
		},
		{
			name:    "new reference is blank",
			oldRef:  existing,
			newRef:  "   ",
			wantErr: domainerrors.ErrInvalidImage,
		},
		{
			name:    "new reference duplicates another image",
			oldRef:  existing,
			newRef:  other,
			wantErr: domainerrors.ErrDuplicateImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := product.ReplaceImage(tt.oldRef, tt.newRef)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ReplaceImage() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
