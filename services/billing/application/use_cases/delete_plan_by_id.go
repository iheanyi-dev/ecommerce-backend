package use_cases

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
)

// DeletePlanByID permanently deletes a plan that has no subscriptions.
type DeletePlanByID struct {
	repository ports.PlanRepository
	telemetry  operationTelemetry
}

// NewDeletePlanByID constructs a plan deletion use case.
func NewDeletePlanByID(
	repository ports.PlanRepository,
	observability ...ports.Observability,
) *DeletePlanByID {
	var dependencies ports.Observability
	if len(observability) > 0 {
		dependencies = observability[0]
	}

	return &DeletePlanByID{
		repository: repository,
		telemetry: newOperationTelemetry(
			dependencies.Logger,
			dependencies.Metrics,
			"delete_plan_by_id",
			"billing_delete_plan_by_id",
		),
	}
}

// Execute deletes the plan identified by id.
func (useCase *DeletePlanByID) Execute(ctx context.Context, id uuid.UUID) error {
	if useCase == nil {
		return application.ErrPlanRepositoryUnavailable
	}

	startedAt := useCase.telemetry.start(ctx)

	if useCase.repository == nil {
		useCase.telemetry.failure(ctx, startedAt, "repository_unavailable")
		return application.ErrPlanRepositoryUnavailable
	}

	if id == uuid.Nil {
		useCase.telemetry.failure(ctx, startedAt, "invalid_input")
		return domain.ErrInvalidPlan
	}

	if err := useCase.repository.DeleteByID(ctx, id); err != nil {
		useCase.telemetry.failure(ctx, startedAt, "deletion")
		return err
	}

	useCase.telemetry.success(ctx, startedAt)
	return nil
}
