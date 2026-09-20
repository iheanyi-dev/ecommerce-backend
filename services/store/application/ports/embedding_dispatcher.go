package ports

import (
	"context"

	"github.com/google/uuid"
)

// StoreEmbedding represents the Store information required by the
// recommendation boundary to generate and store a Store embedding.
//
// The application layer does not know how the embedding is generated or
// where it is persisted.
type StoreEmbedding struct {
	// StoreID identifies the Store whose embedding is being generated.
	StoreID uuid.UUID

	// Name is the Store's current name.
	Name string

	// Description is the Store's current description.
	Description string
}

// EmbeddingDispatcher asynchronously dispatches Store embedding work.
//
// The infrastructure implementation decides how the asynchronous operation
// is delivered to the Recommendation Service.
type EmbeddingDispatcher interface {
	// DispatchStoreEmbedding schedules embedding generation/storage.
	//
	// Failure to dispatch this asynchronous operation does not invalidate
	// successful Store creation.
	DispatchStoreEmbedding(ctx context.Context, embedding StoreEmbedding) error
}
