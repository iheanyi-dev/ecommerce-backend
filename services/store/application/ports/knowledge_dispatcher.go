package ports

import (
	"context"

	"github.com/google/uuid"
)

// KnowledgeIngestionRequested identifies persisted Knowledge metadata that
// the Embedding service should fetch from Storage and process.
type KnowledgeIngestionRequested struct {
	KnowledgeID      uuid.UUID
	StoreID          uuid.UUID
	StorageReference string
}

// KnowledgeIngestionDispatcher publishes Knowledge ingestion requests.
type KnowledgeIngestionDispatcher interface {
	DispatchKnowledgeIngestion(
		ctx context.Context,
		request KnowledgeIngestionRequested,
	) error
}
