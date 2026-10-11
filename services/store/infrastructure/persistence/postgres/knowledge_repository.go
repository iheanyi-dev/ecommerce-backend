package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres/generated"
)

// KnowledgeRepository persists Store knowledge metadata in PostgreSQL.
type KnowledgeRepository struct {
	queries *generated.Queries
}

// NewKnowledgeRepository constructs a KnowledgeRepository.
func NewKnowledgeRepository(queries *generated.Queries) *KnowledgeRepository {
	return &KnowledgeRepository{queries: queries}
}

// GetByStoreID retrieves all knowledge records belonging to a Store.
func (r *KnowledgeRepository) GetByStoreID(
	ctx context.Context,
	storeID uuid.UUID,
) ([]*entities.Knowledge, error) {
	rows, err := r.queries.GetKnowledgeByStoreID(ctx, pgtype.UUID{Bytes: storeID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("get knowledge by store ID: %w", err)
	}

	knowledge := make([]*entities.Knowledge, 0, len(rows))
	for _, row := range rows {
		entity, err := entities.ReconstituteKnowledge(
			uuid.UUID(row.ID.Bytes),
			uuid.UUID(row.StoreID.Bytes),
			row.Filename,
			row.ContentType,
		)
		if err != nil {
			return nil, fmt.Errorf("reconstitute knowledge %s: %w", row.ID, err)
		}

		// The reference is deterministic from StoreID and Filename.
		// Fail rather than silently accepting inconsistent persisted metadata.
		if entity.StorageReference() != row.StorageReference {
			return nil, fmt.Errorf(
				"reconstitute knowledge %s: persisted storage reference mismatch",
				row.ID,
			)
		}

		knowledge = append(knowledge, entity)
	}

	return knowledge, nil
}

// Save persists knowledge metadata. The database enforces storage-reference uniqueness.
func (r *KnowledgeRepository) Save(
	ctx context.Context,
	knowledge *entities.Knowledge,
) error {
	if knowledge == nil {
		return fmt.Errorf("save knowledge: knowledge is nil")
	}

	err := r.queries.SaveKnowledge(ctx, generated.SaveKnowledgeParams{
		ID:               pgtype.UUID{Bytes: knowledge.ID(), Valid: true},
		StoreID:          pgtype.UUID{Bytes: knowledge.StoreID(), Valid: true},
		Filename:         knowledge.Filename(),
		ContentType:      knowledge.ContentType(),
		StorageReference: knowledge.StorageReference(),
	})
	if err != nil {
		return fmt.Errorf("save knowledge: %w", err)
	}

	return nil
}

// DeleteByID deletes knowledge metadata by its identifier.
// Deleting a nonexistent identifier is treated as successful and idempotent.
func (r *KnowledgeRepository) DeleteByID(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := r.queries.DeleteKnowledgeByID(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return fmt.Errorf("delete knowledge by ID: %w", err)
	}

	return nil
}
