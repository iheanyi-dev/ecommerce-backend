package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
)

func TestCreatePlan(t *testing.T) {
	repository := &fakePlanRepository{}
	useCase := use_cases.NewCreatePlan(repository)

	plan, err := useCase.Execute(context.Background(), validCreatePlanInput())
	requireNoError(t, err)

	if plan.ID == uuid.Nil {
		t.Fatal("expected generated plan ID")
	}
	if repository.created == nil {
		t.Fatal("expected plan to be persisted")
	}
	if repository.created.ID != plan.ID {
		t.Fatal("persisted plan ID does not match returned plan")
	}
	if plan.CreatedAt.IsZero() || plan.UpdatedAt.IsZero() {
		t.Fatal("expected creation and update timestamps")
	}
}

func TestCreatePlanRejectsInvalidPlan(t *testing.T) {
	repository := &fakePlanRepository{}
	useCase := use_cases.NewCreatePlan(repository)

	input := validCreatePlanInput()
	input.Name = ""

	_, err := useCase.Execute(context.Background(), input)
	if !errors.Is(err, domain.ErrInvalidPlan) {
		t.Fatalf("expected ErrInvalidPlan, got %v", err)
	}
	if repository.created != nil {
		t.Fatal("invalid plan must not be persisted")
	}
}

func TestCreatePlanPropagatesRepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	repository := &fakePlanRepository{createErr: repositoryErr}
	useCase := use_cases.NewCreatePlan(repository)

	_, err := useCase.Execute(context.Background(), validCreatePlanInput())
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestCreatePlanRejectsNilRepository(t *testing.T) {
	useCase := use_cases.NewCreatePlan(nil)

	_, err := useCase.Execute(context.Background(), validCreatePlanInput())
	if !errors.Is(err, application.ErrPlanRepositoryUnavailable) {
		t.Fatalf("expected ErrPlanRepositoryUnavailable, got %v", err)
	}
}
