package use_cases

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// GetPlanByID retrieves a plan regardless of its availability.
type GetPlanByID struct {
	repository ports.PlanRepository
	telemetry  operationTelemetry
}

// NewGetPlanByID constructs a plan retrieval use case.
func NewGetPlanByID(
	repository ports.PlanRepository,
	observability ...ports.Observability,
) *GetPlanByID {
	var dependencies ports.Observability
	if len(observability) > 0 {
		dependencies = observability[0]
	}

	return &GetPlanByID{
		repository: repository,
		telemetry: newOperationTelemetry(
			dependencies.Logger,
			dependencies.Metrics,
			"get_plan_by_id",
			"billing_get_plan_by_id",
		),
	}
}

// Execute retrieves the plan identified by id.
func (useCase *GetPlanByID) Execute(
	ctx context.Context,
	id uuid.UUID,
) (entities.Plan, error) {
	if useCase == nil {
		return entities.Plan{}, application.ErrPlanRepositoryUnavailable
	}

	startedAt := useCase.telemetry.start(ctx)

	if useCase.repository == nil {
		useCase.telemetry.failure(ctx, startedAt, "repository_unavailable")
		return entities.Plan{}, application.ErrPlanRepositoryUnavailable
	}

	if id == uuid.Nil {
		useCase.telemetry.failure(ctx, startedAt, "invalid_input")
		return entities.Plan{}, domain.ErrInvalidPlan
	}

	plan, err := useCase.repository.GetByID(ctx, id)
	if err != nil {
		useCase.telemetry.failure(ctx, startedAt, "retrieval")
		return entities.Plan{}, err
	}

	useCase.telemetry.success(ctx, startedAt)
	return plan, nil
}
