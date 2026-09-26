package events

import "github.com/google/uuid"

const (
	StoreImageRequestedTopic         = "store.image.requested"
	StoreImageDeletionRequestedTopic = "store.image.deletion.requested"
)

// StoreImageRequested is the JSON contract published by Store when a new
// store image must be persisted by Storage.
type StoreImageRequested struct {
	StoreID    uuid.UUID `json:"store_id"`
	Reference  string    `json:"reference"`
	ContentType string   `json:"content_type"`
	Content    []byte    `json:"content"`
}

// StoreImageDeletionRequested is the JSON contract published by Store when
// an existing store image must be deleted by Storage.
type StoreImageDeletionRequested struct {
	StoreID   uuid.UUID `json:"store_id"`
	Reference string    `json:"reference"`
}
