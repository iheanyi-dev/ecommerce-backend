package image

import (
	"context"
	"strings"

	infraerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/errors"
)

// ImageFormatResolver resolves image Content-Types to canonical file
// extensions.
//
// The resolver deliberately contains only format mapping logic. It does not
// inspect image bytes and does not depend on HTTP or multipart packages.
type ImageFormatResolver struct{}

// NewImageFormatResolver constructs the Store image format resolver.
func NewImageFormatResolver() *ImageFormatResolver {
	return &ImageFormatResolver{}
}

// ResolveExtension resolves an image Content-Type to its canonical extension.
func (r *ImageFormatResolver) ResolveExtension(
	ctx context.Context,
	contentType string,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg":
		return "jpg", nil
	case "image/png":
		return "png", nil
	case "image/webp":
		return "webp", nil
	case "image/gif":
		return "gif", nil
	case "image/bmp":
		return "bmp", nil
	case "image/tiff":
		return "tiff", nil
	case "image/svg+xml":
		return "svg", nil
	default:
		return "", infraerrors.ErrUnsupportedImageFormat
	}
}
