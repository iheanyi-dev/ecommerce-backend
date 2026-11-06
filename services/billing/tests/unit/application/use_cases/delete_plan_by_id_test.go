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

func TestDeletePlanByID(t *testing.T) {
	repository := &fakePlanRepository{}
	useCase := use_cases.NewDeletePlanByID(repository)
	id := uuid.New()

	requireNoError(t, useCase.Execute(context.Background(), id))

	if repository.deletedID != id {
		t.Fatal("expected repository to receive the plan ID")
	}
}

func TestDeletePlanByIDRejectsNilID(t *testing.T) {
	repository := &fakePlanRepository{}
	useCase := use_cases.NewDeletePlanByID(repository)

	err := useCase.Execute(context.Background(), uuid.Nil)
	if !errors.Is(err, domain.ErrInvalidPlan) {
		t.Fatalf("expected ErrInvalidPlan, got %v", err)
	}
	if repository.deletedID != uuid.Nil {
		t.Fatal("repository must not be called with a nil ID")
	}
}

func TestDeletePlanByIDPropagatesPlanInUseError(t *testing.T) {
	repository := &fakePlanRepository{deleteErr: application.ErrPlanInUse}
	useCase := use_cases.NewDeletePlanByID(repository)

	err := useCase.Execute(context.Background(), uuid.New())
	if !errors.Is(err, application.ErrPlanInUse) {
		t.Fatalf("expected ErrPlanInUse, got %v", err)
	}
}

func TestDeletePlanByIDRejectsNilRepository(t *testing.T) {
	useCase := use_cases.NewDeletePlanByID(nil)

	err := useCase.Execute(context.Background(), uuid.New())
	if !errors.Is(err, application.ErrPlanRepositoryUnavailable) {
		t.Fatalf("expected ErrPlanRepositoryUnavailable, got %v", err)
	}
}
