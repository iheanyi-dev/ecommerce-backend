package use_cases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

// UpdatePlanInput contains the complete replacement plan details.
type UpdatePlanInput struct {
	Type         string
	Name         string
	Description  string
	Price        value_objects.Money
	Duration     time.Duration
	Capabilities entities.PlanCapabilities
	Available    bool
	Promotion    *entities.Promotion
}

// UpdatePlan replaces all administrator-editable plan details.
type UpdatePlan struct {
	repository ports.PlanRepository
	telemetry  operationTelemetry
}

// NewUpdatePlan constructs a plan update use case.
func NewUpdatePlan(
	repository ports.PlanRepository,
	observability ...ports.Observability,
) *UpdatePlan {
	var dependencies ports.Observability
	if len(observability) > 0 {
		dependencies = observability[0]
	}

	return &UpdatePlan{
		repository: repository,
		telemetry: newOperationTelemetry(
			dependencies.Logger,
			dependencies.Metrics,
			"update_plan",
			"billing_update_plan",
		),
	}
}

// Execute replaces the administrator-editable details of an existing plan.
func (useCase *UpdatePlan) Execute(
	ctx context.Context,
	id uuid.UUID,
	input UpdatePlanInput,
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

	existing, err := useCase.repository.GetByID(ctx, id)
	if err != nil {
		useCase.telemetry.failure(ctx, startedAt, "retrieval")
		return entities.Plan{}, err
	}

	updated := entities.Plan{
		ID:           existing.ID,
		Type:         input.Type,
		Name:         input.Name,
		Description:  input.Description,
		Price:        input.Price,
		Duration:     input.Duration,
		Capabilities: input.Capabilities,
		Available:    input.Available,
		Promotion:    input.Promotion,
		CreatedAt:    existing.CreatedAt,
		UpdatedAt:    time.Now().UTC(),
	}

	if err := updated.Validate(); err != nil {
		useCase.telemetry.failure(ctx, startedAt, "validation")
		return entities.Plan{}, err
	}

	if err := useCase.repository.Update(ctx, updated); err != nil {
		useCase.telemetry.failure(ctx, startedAt, "persistence")
		return entities.Plan{}, err
	}

	useCase.telemetry.success(ctx, startedAt)
	return updated, nil
}
