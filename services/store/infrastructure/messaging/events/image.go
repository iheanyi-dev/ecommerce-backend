package events

import (
	"github.com/google/uuid"
)

const (
	StoreImageRequestedTopic         = "store.image.requested"
	StoreImageDeletionRequestedTopic = "store.image.deletion.requested"
)

// StoreImageRequested represents an asynchronous request to persist a Store
// image in the Storage service.
//
// The event contains the complete image payload because the current
// ImageStorageDispatcher contract owns the image bytes.
type StoreImageRequested struct {
	StoreID     uuid.UUID `json:"store_id"`
	Reference   string    `json:"reference"`
	ContentType string    `json:"content_type"`
	Content     []byte    `json:"content"`
}

// StoreImageDeletionRequested represents an asynchronous request to remove
// an obsolete Store image from Storage.
type StoreImageDeletionRequested struct {
	StoreID   uuid.UUID `json:"store_id"`
	Reference string    `json:"reference"`
}
