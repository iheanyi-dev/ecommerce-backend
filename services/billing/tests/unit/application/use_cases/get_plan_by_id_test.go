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

func TestGetPlanByIDReturnsUnavailablePlan(t *testing.T) {
	plan := validPlan()
	plan.Available = false

	repository := &fakePlanRepository{plan: plan}
	useCase := use_cases.NewGetPlanByID(repository)

	result, err := useCase.Execute(context.Background(), plan.ID)
	requireNoError(t, err)

	if result.ID != plan.ID || result.Available {
		t.Fatal("expected the unavailable plan to be returned unchanged")
	}
}

func TestGetPlanByIDRejectsNilID(t *testing.T) {
	repository := &fakePlanRepository{}
	useCase := use_cases.NewGetPlanByID(repository)

	_, err := useCase.Execute(context.Background(), uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidPlan) {
		t.Fatalf("expected ErrInvalidPlan, got %v", err)
	}
	if repository.getByIDCall != 0 {
		t.Fatal("repository must not be called with a nil ID")
	}
}

func TestGetPlanByIDPropagatesRepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	repository := &fakePlanRepository{getByIDErr: repositoryErr}
	useCase := use_cases.NewGetPlanByID(repository)

	_, err := useCase.Execute(context.Background(), uuid.New())
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestGetPlanByIDRejectsNilRepository(t *testing.T) {
	useCase := use_cases.NewGetPlanByID(nil)

	_, err := useCase.Execute(context.Background(), uuid.New())
	if !errors.Is(err, application.ErrPlanRepositoryUnavailable) {
		t.Fatalf("expected ErrPlanRepositoryUnavailable, got %v", err)
	}
}
