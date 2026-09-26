package dto

import "io"

// GetPublicObjectInput identifies a public object using the complete
// reference supplied by the calling layer.
type GetPublicObjectInput struct {
	Reference string
}

// GetPublicObjectOutput contains the streamed public object and its MIME type.
type GetPublicObjectOutput struct {
	Content     io.ReadCloser
	ContentType string
}
