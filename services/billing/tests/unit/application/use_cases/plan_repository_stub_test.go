package use_cases_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

type fakePlanRepository struct {
	plans       []entities.Plan
	plan        entities.Plan
	createErr   error
	updateErr   error
	deleteErr   error
	getAllErr   error
	getByIDErr  error
	created     *entities.Plan
	updated     *entities.Plan
	deletedID   uuid.UUID
	getByIDCall int
}

var _ ports.PlanRepository = (*fakePlanRepository)(nil)

func (repository *fakePlanRepository) Create(
	_ context.Context,
	plan entities.Plan,
) error {
	if repository.createErr != nil {
		return repository.createErr
	}
	repository.created = &plan
	return nil
}

func (repository *fakePlanRepository) Update(
	_ context.Context,
	plan entities.Plan,
) error {
	if repository.updateErr != nil {
		return repository.updateErr
	}
	repository.updated = &plan
	return nil
}

func (repository *fakePlanRepository) DeleteByID(
	_ context.Context,
	id uuid.UUID,
) error {
	if repository.deleteErr != nil {
		return repository.deleteErr
	}
	repository.deletedID = id
	return nil
}

func (repository *fakePlanRepository) GetAll(
	_ context.Context,
) ([]entities.Plan, error) {
	if repository.getAllErr != nil {
		return nil, repository.getAllErr
	}
	return repository.plans, nil
}

func (repository *fakePlanRepository) GetByID(
	_ context.Context,
	_ uuid.UUID,
) (entities.Plan, error) {
	repository.getByIDCall++
	if repository.getByIDErr != nil {
		return entities.Plan{}, repository.getByIDErr
	}
	return repository.plan, nil
}
