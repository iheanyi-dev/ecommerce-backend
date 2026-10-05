package dto

// HeadObjectInput identifies an object using the complete reference supplied
// by the calling service.
type HeadObjectInput struct {
	Reference string
}

// HeadObjectOutput contains metadata for an existing object.
//
// The absence of an error establishes that the object exists.
type HeadObjectOutput struct {
	ContentType string
}
