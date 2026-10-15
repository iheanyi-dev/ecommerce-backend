package ports

import (
	"context"
	"io"
)

// StorageClient exposes object operations required by Product.
// Product generates the storage reference; Storage must use the supplied one.
type StorageClient interface {
	PutObject(ctx context.Context, reference, contentType string, content io.Reader) error
	DeleteObject(ctx context.Context, reference string) error
}
