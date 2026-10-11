package ports

import (
	"context"

	"github.com/google/uuid"
)

type KnowledgeDeletionRequested struct {
	KnowledgeID uuid.UUID
	StoreID     uuid.UUID
}

type KnowledgeDeletionDispatcher interface {
	DispatchKnowledgeDeletion(ctx context.Context, request KnowledgeDeletionRequested) error
}
