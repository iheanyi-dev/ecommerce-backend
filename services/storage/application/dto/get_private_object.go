package dto

import "io"

// GetPrivateObjectInput identifies a private object using the complete
// reference supplied by the calling service.
type GetPrivateObjectInput struct {
	Reference string
}

// GetPrivateObjectOutput contains the streamed private object and its MIME
// type.
type GetPrivateObjectOutput struct {
	Content     io.ReadCloser
	ContentType string
}
