package use_cases

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

// GetPlans retrieves all plans, including unavailable plans.
type GetPlans struct {
	repository ports.PlanRepository
	telemetry  operationTelemetry
}

// NewGetPlans constructs a plan listing use case.
func NewGetPlans(
	repository ports.PlanRepository,
	observability ...ports.Observability,
) *GetPlans {
	var dependencies ports.Observability
	if len(observability) > 0 {
		dependencies = observability[0]
	}

	return &GetPlans{
		repository: repository,
		telemetry: newOperationTelemetry(
			dependencies.Logger,
			dependencies.Metrics,
			"get_plans",
			"billing_get_plans",
		),
	}
}

// Execute retrieves all subscription plans.
func (useCase *GetPlans) Execute(ctx context.Context) ([]entities.Plan, error) {
	if useCase == nil {
		return nil, application.ErrPlanRepositoryUnavailable
	}

	startedAt := useCase.telemetry.start(ctx)

	if useCase.repository == nil {
		useCase.telemetry.failure(ctx, startedAt, "repository_unavailable")
		return nil, application.ErrPlanRepositoryUnavailable
	}

	plans, err := useCase.repository.GetAll(ctx)
	if err != nil {
		useCase.telemetry.failure(ctx, startedAt, "retrieval")
		return nil, err
	}

	if plans == nil {
		plans = []entities.Plan{}
	}

	useCase.telemetry.success(ctx, startedAt)
	return plans, nil
}
