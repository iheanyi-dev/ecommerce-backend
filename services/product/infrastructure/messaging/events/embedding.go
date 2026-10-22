package events

import "github.com/google/uuid"

const (
	ProductEmbeddingRequestedTopic         = "product.embedding.requested"
	ProductEmbeddingDeletionRequestedTopic = "product.embedding.deletion.requested"
)

// ProductEmbeddingRequested asks the Embedding service to create or update a vector.
type ProductEmbeddingRequested struct {
	ProductID   uuid.UUID `json:"product_id"`
	StoreID     uuid.UUID `json:"store_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// ProductEmbeddingDeletionRequested asks the Embedding service to delete a vector.
type ProductEmbeddingDeletionRequested struct {
	ProductID uuid.UUID `json:"product_id"`
	StoreID   uuid.UUID `json:"store_id"`
}
