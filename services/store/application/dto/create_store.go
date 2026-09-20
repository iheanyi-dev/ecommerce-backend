package dto

// StoreImageInput represents an image supplied when creating a Store.
//
// The application layer receives this technology-neutral representation
// rather than a presentation-specific multipart file.
type StoreImageInput struct {
	// Content contains the image bytes.
	Content []byte

	// Filename is the original filename supplied with the image.
	Filename string

	// ContentType identifies the image media type.
	ContentType string
}

// CreateStoreInput contains the data required to create a Store.
//
// Owner identity is deliberately not included here. The authenticated
// application identity is supplied separately by the use-case boundary.
type CreateStoreInput struct {
	Name        string
	Slug        string
	Description string

	// Image is nil when the Store is created without an image.
	Image *StoreImageInput
}

// CreateStoreOutput represents the Store created by the application layer.
type CreateStoreOutput = StoreOutput
