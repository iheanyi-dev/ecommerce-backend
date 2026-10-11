package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
)

// KnowledgeRepository defines persistence operations for Store knowledge metadata.
type KnowledgeRepository interface {
	GetByStoreID(ctx context.Context, storeID uuid.UUID) ([]*entities.Knowledge, error)
	Save(ctx context.Context, knowledge *entities.Knowledge) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
}
