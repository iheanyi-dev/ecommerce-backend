package image_test

import (
	"context"
	"errors"
	"testing"

	infraerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/image"
)

func TestImageFormatResolver_ResolveExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{"jpeg", "image/jpeg", "jpg"},
		{"png", "image/png", "png"},
		{"webp", "image/webp", "webp"},
		{"gif", "image/gif", "gif"},
		{"bmp", "image/bmp", "bmp"},
		{"tiff", "image/tiff", "tiff"},
		{"svg", "image/svg+xml", "svg"},
	}

	resolver := image.NewImageFormatResolver()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolver.ResolveExtension(
				context.Background(),
				tt.contentType,
			)
			if err != nil {
				t.Fatalf("ResolveExtension() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf(
					"ResolveExtension() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestImageFormatResolver_ResolveExtension_UnsupportedContentType(t *testing.T) {
	t.Parallel()

	resolver := image.NewImageFormatResolver()

	_, err := resolver.ResolveExtension(
		context.Background(),
		"application/pdf",
	)
	if err == nil {
		t.Fatal("ResolveExtension() expected error, got nil")
	}

	if !errors.Is(err, infraerrors.ErrUnsupportedImageFormat) {
		t.Fatalf("ResolveExtension() error = %v, want ErrUnsupportedImageFormat", err)
	}
}

func TestImageFormatResolver_ResolveExtension_EmptyContentType(t *testing.T) {
	t.Parallel()

	resolver := image.NewImageFormatResolver()

	_, err := resolver.ResolveExtension(
		context.Background(),
		"",
	)
	if err == nil {
		t.Fatal("ResolveExtension() expected error, got nil")
	}
}
