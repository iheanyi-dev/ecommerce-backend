package use_cases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

// CreatePlanInput contains the administrator-supplied plan details.
type CreatePlanInput struct {
	Type         string
	Name         string
	Description  string
	Price        value_objects.Money
	Duration     time.Duration
	Capabilities entities.PlanCapabilities
	Available    bool
	Promotion    *entities.Promotion
}

// CreatePlan creates a new subscription plan.
type CreatePlan struct {
	repository ports.PlanRepository
	telemetry  operationTelemetry
}

// NewCreatePlan constructs a plan creation use case.
// Observability dependencies are optional.
func NewCreatePlan(
	repository ports.PlanRepository,
	observability ...ports.Observability,
) *CreatePlan {
	var dependencies ports.Observability
	if len(observability) > 0 {
		dependencies = observability[0]
	}

	return &CreatePlan{
		repository: repository,
		telemetry: newOperationTelemetry(
			dependencies.Logger,
			dependencies.Metrics,
			"create_plan",
			"billing_create_plan",
		),
	}
}

// Execute validates and persists a new subscription plan.
func (useCase *CreatePlan) Execute(
	ctx context.Context,
	input CreatePlanInput,
) (entities.Plan, error) {
	if useCase == nil {
		return entities.Plan{}, application.ErrPlanRepositoryUnavailable
	}

	startedAt := useCase.telemetry.start(ctx)

	if useCase.repository == nil {
		useCase.telemetry.failure(ctx, startedAt, "repository_unavailable")
		return entities.Plan{}, application.ErrPlanRepositoryUnavailable
	}

	now := time.Now().UTC()

	plan := entities.Plan{
		ID:           uuid.New(),
		Type:         input.Type,
		Name:         input.Name,
		Description:  input.Description,
		Price:        input.Price,
		Duration:     input.Duration,
		Capabilities: input.Capabilities,
		Available:    input.Available,
		Promotion:    input.Promotion,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := plan.Validate(); err != nil {
		useCase.telemetry.failure(ctx, startedAt, "validation")
		return entities.Plan{}, err
	}

	if err := useCase.repository.Create(ctx, plan); err != nil {
		useCase.telemetry.failure(ctx, startedAt, "persistence")
		return entities.Plan{}, err
	}

	useCase.telemetry.success(ctx, startedAt)
	return plan, nil
}
