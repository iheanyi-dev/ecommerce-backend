package events

import "github.com/google/uuid"

const (
	StoreEmbeddingRequestedTopic         = "store.embedding.requested"
	StoreEmbeddingDeletionRequestedTopic = "store.embedding.deletion.requested"
)

// StoreEmbeddingRequested represents a request for the Recommendation
// Service to generate/update a Store embedding.
type StoreEmbeddingRequested struct {
	StoreID     uuid.UUID `json:"store_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// StoreEmbeddingDeletionRequested represents a request for the
// Recommendation Service to remove a Store embedding.
type StoreEmbeddingDeletionRequested struct {
	StoreID uuid.UUID `json:"store_id"`
}
