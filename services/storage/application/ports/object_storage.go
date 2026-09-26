package ports

import (
	"context"
	"io"
)

// ObjectStorage is the generic storage backend abstraction.
//
// Storage exposes generic object operations only. Business services own
// object references, business validation, authorization, and the meaning
// of each object.
type ObjectStorage interface {
	PutObject(
		ctx context.Context,
		reference string,
		content io.Reader,
		contentType string,
	) error

	GetPublicObject(
		ctx context.Context,
		reference string,
	) (io.ReadCloser, string, error)

	GetPrivateObject(
		ctx context.Context,
		reference string,
	) (io.ReadCloser, string, error)

	DeleteObject(
		ctx context.Context,
		reference string,
	) error
}
