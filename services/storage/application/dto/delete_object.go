package dto

// DeleteObjectInput identifies the object to delete using the complete
// reference supplied by the calling service.
type DeleteObjectInput struct {
	Reference string
}
