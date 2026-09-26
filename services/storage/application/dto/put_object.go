package dto

import "io"

// PutObjectInput contains the generic object data supplied by an internal
// calling service.
//
// Storage deliberately does not know the business meaning of the object.
// The calling service owns the reference/key and is responsible for all
// business-level validation before invoking Storage.
type PutObjectInput struct {
	// Reference is the complete object key chosen by the calling service.
	Reference string

	// Content is the object body. It is streamed rather than loaded into
	// memory by the application layer.
	Content io.Reader

	// ContentType is the MIME type supplied by the calling service.
	ContentType string
}
