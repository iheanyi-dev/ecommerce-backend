package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// RemoveStaleKnowledgeService requests removal of a specific knowledge file
// from the AI service after Store authorization.
type RemoveStaleKnowledgeService interface {
	Execute(ctx context.Context, input dto.RemoveStaleKnowledgeInput) (dto.RemoveStaleKnowledgeOutput, error)
}
