package ports

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// AddStoreKnowledgeService submits a Store knowledge file to the
// configured AI knowledge-processing service after Store authorization.
type AddStoreKnowledgeService interface {
	Execute(ctx context.Context, input dto.AddStoreKnowledgeInput) (dto.AddStoreKnowledgeOutput, error)
}
