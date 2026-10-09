package ports

import (
	"context"

	"github.com/google/uuid"
)

// StoreEmbedding represents the Store information required by the
// recommendation boundary to generate and store a Store embedding.
type StoreEmbedding struct {
	StoreID     uuid.UUID
	Name        string
	Description string
}

// EmbeddingDispatcher asynchronously dispatches Store embedding work.
type EmbeddingDispatcher interface {
	// DispatchStoreEmbedding schedules embedding generation/storage.
	DispatchStoreEmbedding(ctx context.Context, embedding StoreEmbedding) error

	// DispatchStoreEmbeddingDeletion schedules removal of the Store embedding.
	DispatchStoreEmbeddingDeletion(ctx context.Context, storeID uuid.UUID) error
}
