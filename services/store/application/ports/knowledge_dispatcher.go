package ports

import (
	"context"

	"github.com/google/uuid"
)

// StoreKnowledge represents a knowledge file that should be processed and
// associated with a Store by the AI service.
//
// Store owns the Store identity and forwards the supplied file. The AI
// service owns file processing, knowledge extraction, embeddings, and vector
// storage.
type StoreKnowledge struct {
	StoreID     uuid.UUID
	Content     []byte
	Filename    string
	ContentType string
}

// KnowledgeDispatcher sends a Store knowledge file to the AI service.
//
// The application layer does not know how the file is processed or how the
// resulting knowledge is persisted.
type KnowledgeDispatcher interface {
	DispatchStoreKnowledge(ctx context.Context, knowledge StoreKnowledge) error
}
