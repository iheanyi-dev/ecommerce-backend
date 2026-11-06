package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

func TestUpdatePlan(t *testing.T) {
	existing := validPlan()
	repository := &fakePlanRepository{plan: existing}
	useCase := use_cases.NewUpdatePlan(repository)

	input := validUpdatePlanInput()
	input.Name = "Premium Plus"
	input.Description = "Updated description"
	input.Duration = 60 * 24 * time.Hour
	input.Capabilities = entities.PlanCapabilities{
		MaxProducts: 10,
		AIEnabled:   false,
	}
	input.Available = false

	updated, err := useCase.Execute(context.Background(), existing.ID, input)
	requireNoError(t, err)

	if repository.updated == nil {
		t.Fatal("expected updated plan to be persisted")
	}
	if updated.ID != existing.ID {
		t.Fatal("update must preserve plan ID")
	}
	if !updated.CreatedAt.Equal(existing.CreatedAt) {
		t.Fatal("update must preserve CreatedAt")
	}
	if !updated.UpdatedAt.After(existing.UpdatedAt) {
		t.Fatal("expected UpdatedAt to advance")
	}
	if updated.Name != input.Name ||
		updated.Description != input.Description ||
		updated.Duration != input.Duration ||
		updated.Capabilities != input.Capabilities ||
		updated.Available != input.Available {
		t.Fatal("updated plan does not contain all replacement fields")
	}
}

func TestUpdatePlanClearsPromotion(t *testing.T) {
	existing := validPlan()
	existing.Promotion = &entities.Promotion{
		Name:                "Launch",
		DiscountAmountMinor: 500,
		StartsAt:            time.Now().UTC(),
		EndsAt:              time.Now().UTC().Add(24 * time.Hour),
	}
	repository := &fakePlanRepository{plan: existing}
	useCase := use_cases.NewUpdatePlan(repository)

	input := validUpdatePlanInput()
	input.Promotion = nil

	updated, err := useCase.Execute(context.Background(), existing.ID, input)
	requireNoError(t, err)

	if updated.Promotion != nil {
		t.Fatal("expected nil promotion to clear the existing promotion")
	}
}

func TestUpdatePlanRejectsNilID(t *testing.T) {
	repository := &fakePlanRepository{}
	useCase := use_cases.NewUpdatePlan(repository)

	_, err := useCase.Execute(context.Background(), uuid.Nil, validUpdatePlanInput())
	if !errors.Is(err, domain.ErrInvalidPlan) {
		t.Fatalf("expected ErrInvalidPlan, got %v", err)
	}
	if repository.getByIDCall != 0 {
		t.Fatal("repository must not be called with a nil ID")
	}
}

func TestUpdatePlanRejectsInvalidReplacement(t *testing.T) {
	existing := validPlan()
	repository := &fakePlanRepository{plan: existing}
	useCase := use_cases.NewUpdatePlan(repository)

	input := validUpdatePlanInput()
	input.Name = ""

	_, err := useCase.Execute(context.Background(), existing.ID, input)
	if !errors.Is(err, domain.ErrInvalidPlan) {
		t.Fatalf("expected ErrInvalidPlan, got %v", err)
	}
	if repository.updated != nil {
		t.Fatal("invalid replacement must not be persisted")
	}
}

func TestUpdatePlanPropagatesGetError(t *testing.T) {
	repositoryErr := errors.New("plan not found")
	repository := &fakePlanRepository{getByIDErr: repositoryErr}
	useCase := use_cases.NewUpdatePlan(repository)

	_, err := useCase.Execute(context.Background(), uuid.New(), validUpdatePlanInput())
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestUpdatePlanPropagatesUpdateError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	repository := &fakePlanRepository{
		plan:      validPlan(),
		updateErr: repositoryErr,
	}
	useCase := use_cases.NewUpdatePlan(repository)

	_, err := useCase.Execute(
		context.Background(),
		repository.plan.ID,
		validUpdatePlanInput(),
	)
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestUpdatePlanRejectsNilRepository(t *testing.T) {
	useCase := use_cases.NewUpdatePlan(nil)

	_, err := useCase.Execute(context.Background(), uuid.New(), validUpdatePlanInput())
	if !errors.Is(err, application.ErrPlanRepositoryUnavailable) {
		t.Fatalf("expected ErrPlanRepositoryUnavailable, got %v", err)
	}
}
