package ports

import "context"

// ImageFormatResolver resolves a supported image content type to its
// canonical file extension.
//
// The application layer depends on this capability rather than on a
// particular MIME/image library. Infrastructure provides the implementation.
type ImageFormatResolver interface {
	// ResolveExtension returns the canonical extension for the supplied
	// image Content-Type.
	//
	// For example, an implementation may resolve:
	//   image/jpeg -> jpg
	//   image/png  -> png
	//   image/webp -> webp
	//
	// Unsupported content types must return an error.
	ResolveExtension(ctx context.Context, contentType string) (string, error)
}
