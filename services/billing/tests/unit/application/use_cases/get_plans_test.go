package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

func TestGetPlansReturnsAllPlans(t *testing.T) {
	available := validPlan()
	unavailable := validPlan()
	unavailable.ID = available.ID
	unavailable.ID = unavailable.ID
	unavailable.Available = false

	repository := &fakePlanRepository{
		plans: []entities.Plan{available, unavailable},
	}
	useCase := use_cases.NewGetPlans(repository)

	plans, err := useCase.Execute(context.Background())
	requireNoError(t, err)

	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}
	if plans[0].Available != true || plans[1].Available != false {
		t.Fatal("expected both available and unavailable plans")
	}
}

func TestGetPlansReturnsEmptySliceWhenRepositoryReturnsNil(t *testing.T) {
	repository := &fakePlanRepository{}
	useCase := use_cases.NewGetPlans(repository)

	plans, err := useCase.Execute(context.Background())
	requireNoError(t, err)

	if plans == nil {
		t.Fatal("expected an empty slice, got nil")
	}
	if len(plans) != 0 {
		t.Fatalf("expected 0 plans, got %d", len(plans))
	}
}

func TestGetPlansPropagatesRepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	repository := &fakePlanRepository{getAllErr: repositoryErr}
	useCase := use_cases.NewGetPlans(repository)

	_, err := useCase.Execute(context.Background())
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestGetPlansRejectsNilRepository(t *testing.T) {
	useCase := use_cases.NewGetPlans(nil)

	_, err := useCase.Execute(context.Background())
	if !errors.Is(err, application.ErrPlanRepositoryUnavailable) {
		t.Fatalf("expected ErrPlanRepositoryUnavailable, got %v", err)
	}
}
