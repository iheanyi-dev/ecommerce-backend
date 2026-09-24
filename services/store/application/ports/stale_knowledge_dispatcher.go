package ports

import (
	"context"

	"github.com/google/uuid"
)

// StaleKnowledge represents a request to remove one knowledge file associated
// with a Store.
//
// The AI service owns the knowledge file and its persistence. Store only
// forwards the Store identity and file name after authorization.
type StaleKnowledge struct {
	StoreID  uuid.UUID
	Filename string
}

// StaleKnowledgeDispatcher sends a request to the AI service to remove a
// specific Store knowledge file.
type StaleKnowledgeDispatcher interface {
	DispatchRemoveStaleKnowledge(ctx context.Context, knowledge StaleKnowledge) error
}
