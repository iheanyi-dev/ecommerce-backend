package fixtures

import (
	"context"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
)

type MockStoreRepository struct {
	CreateCalled bool
	CreateError  error
	OwnerID      uuid.UUID

	DeleteCalled   bool
	DeletedStoreID uuid.UUID
	DeleteError    error

	ChangePlanCalled  bool
	ChangePlanStoreID uuid.UUID
	ChangedPlanStore  *entities.Store
	ChangePlanErr     error
	Store             *entities.Store

	FindByIDCalled bool
	FindByIDErr    error
	FindByIDValue  uuid.UUID

	FindByOwnerIDCalled bool
	FindByOwnerIDErr    error

	FindBySlugErr error

	ChangeStatusCalled  bool
	ChangeStatusStoreID uuid.UUID
	ChangeStatusValue   string
	ChangeStatusErr     error
}

func (f *MockStoreRepository) Create(
	_ context.Context,
	store *entities.Store,
) error {
	f.CreateCalled = true
	f.Store = store

	return f.CreateError
}

// func (f *MockStoreRepository) FindByID(
// 	_ context.Context,
// 	storeID uuid.UUID,
// ) (*entities.Store, error) {
// 	f.FindByIDCalled = true

// 	// Keep the requested ID available through the mock's public state.
// 	f.Store = nil

// 	_ = storeID

// 	return nil, f.FindByIDErr
// }

func (f *MockStoreRepository) FindByID(
	_ context.Context,
	storeID uuid.UUID,
) (*entities.Store, error) {
	f.FindByIDCalled = true
	f.FindByIDValue = storeID

	if f.FindByIDErr != nil {
		return nil, f.FindByIDErr
	}

	return f.Store, nil
}

func (f *MockStoreRepository) FindByOwnerID(
	_ context.Context,
	ownerID uuid.UUID,
) (*entities.Store, error) {
	f.FindByOwnerIDCalled = true
	f.OwnerID = ownerID

	if f.FindByOwnerIDErr != nil {
		return nil, f.FindByOwnerIDErr
	}

	return f.Store, nil
}

// func (f *MockStoreRepository) FindByOwnerID(
// 	_ context.Context,
// 	ownerID uuid.UUID,
// ) (*entities.Store, error) {
// 	f.FindByOwnerCalled = true
// 	f.OwnerID = ownerID

// 	return f.Store, nil
// }

func (f *MockStoreRepository) FindBySlug(
	_ context.Context,
	_ string,
) (*entities.Store, error) {
	if f.FindBySlugErr != nil {
		return nil, f.FindBySlugErr
	}
	return f.Store, nil
}

func (f *MockStoreRepository) ExistsBySlug(
	_ context.Context,
	_ string,
) (bool, error) {
	return true, nil
}

func (f *MockStoreRepository) ListActive(
	_ context.Context,
	_ string,
	_ int,
	_ int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *MockStoreRepository) Update(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *MockStoreRepository) Delete(
	_ context.Context,
	storeID uuid.UUID,
) error {
	f.DeleteCalled = true
	f.DeletedStoreID = storeID

	return f.DeleteError
}

func (f *MockStoreRepository) ChangePlan(
	_ context.Context,
	store *entities.Store,
) error {
	f.ChangePlanCalled = true
	f.ChangedPlanStore = store

	if store != nil {
		f.ChangePlanStoreID = store.ID().Value()
	}

	return f.ChangePlanErr
}

func (f *MockStoreRepository) ChangeStatus(
	_ context.Context,
	storeID uuid.UUID,
	status string,
) error {
	f.ChangeStatusCalled = true
	f.ChangeStatusStoreID = storeID
	f.ChangeStatusValue = status

	return f.ChangeStatusErr
}
