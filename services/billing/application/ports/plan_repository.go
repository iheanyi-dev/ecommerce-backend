package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// PlanRepository defines persistence operations for subscription plans.
type PlanRepository interface {
	Create(ctx context.Context, plan entities.Plan) error
	Update(ctx context.Context, plan entities.Plan) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
	GetAll(ctx context.Context) ([]entities.Plan, error)
	GetByID(ctx context.Context, id uuid.UUID) (entities.Plan, error)
}
