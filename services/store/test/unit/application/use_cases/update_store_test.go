package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	appErrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

type fakeUpdateStoreRepository struct {
	store *entities.Store

	findByIDErr     error
	existsBySlug    bool
	existsBySlugErr error
	updateErr       error

	findByIDCalled     bool
	existsBySlugCalled bool
	updateCalled       bool
	lastExistsSlug     string
	lastUpdatedStore   *entities.Store
}

func (f *fakeUpdateStoreRepository) Create(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *fakeUpdateStoreRepository) Delete(
	_ context.Context,
	_ uuid.UUID,
) error {
	return nil
}

func (f *fakeUpdateStoreRepository) FindByID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Store, error) {
	f.findByIDCalled = true

	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}

	return f.store, nil
}

func (f *fakeUpdateStoreRepository) FindByOwnerID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeUpdateStoreRepository) FindBySlug(
	_ context.Context,
	_ string,
) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeUpdateStoreRepository) ListActive(
	_ context.Context,
	_ string,
	_ int,
	_ int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *fakeUpdateStoreRepository) ExistsBySlug(
	_ context.Context,
	slug string,
) (bool, error) {
	f.existsBySlugCalled = true
	f.lastExistsSlug = slug

	if f.existsBySlugErr != nil {
		return false, f.existsBySlugErr
	}

	return f.existsBySlug, nil
}

func (f *fakeUpdateStoreRepository) Update(
	_ context.Context,
	store *entities.Store,
) error {
	f.updateCalled = true
	f.lastUpdatedStore = store

	return f.updateErr
}

type fakeUpdateStoreLogger struct{}

func (f *fakeUpdateStoreLogger) Log(
	_ context.Context,
	_ ports.LogEvent,
) error {
	return nil
}

type fakeUpdateStoreMetrics struct{}

func (f *fakeUpdateStoreMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	return nil
}

func (f *fakeUpdateStoreMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	return nil
}

func TestUpdateStoreUseCaseExecute_UpdatesOwnedStore(t *testing.T) {
	storeID := uuid.New()
	ownerID := uuid.New()

	store := mustReconstituteStore(t, storeID, ownerID)

	repository := &fakeUpdateStoreRepository{
		store: store,
	}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: ownerID,
			Role:   "vendor",
		},
	)

	output, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     storeID,
		Name:        "Updated Store",
		Slug:        "updated-store",
		Description: "Updated description",
	})

	require.NoError(t, err)
	require.Equal(t, storeID, output.ID.Value())
	require.Equal(t, ownerID, output.OwnerID.Value())
	require.Equal(t, "Updated Store", output.Name.Value())
	require.Equal(t, "updated-store", output.Slug.Value())
	require.Equal(t, "Updated description", output.Description.Value())

	require.True(t, repository.findByIDCalled)
	require.True(t, repository.existsBySlugCalled)
	require.Equal(t, "updated-store", repository.lastExistsSlug)
	require.True(t, repository.updateCalled)
	require.Same(t, store, repository.lastUpdatedStore)
}

func TestUpdateStoreUseCaseExecute_RequiresAuthentication(t *testing.T) {
	repository := &fakeUpdateStoreRepository{}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	_, err := useCase.Execute(
		context.Background(),
		dto.UpdateStoreInput{
			StoreID:     uuid.New(),
			Name:        "Updated Store",
			Slug:        "updated-store",
			Description: "Updated description",
		},
	)

	require.ErrorIs(t, err, appErrors.ErrUnauthenticated)
	require.False(t, repository.findByIDCalled)
	require.False(t, repository.updateCalled)
}

func TestUpdateStoreUseCaseExecute_RejectsNonOwner(t *testing.T) {
	storeID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	store := mustReconstituteStore(t, storeID, ownerID)

	repository := &fakeUpdateStoreRepository{
		store: store,
	}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: otherUserID,
			Role:   "vendor",
		},
	)

	_, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     storeID,
		Name:        "Updated Store",
		Slug:        "updated-store",
		Description: "Updated description",
	})

	require.ErrorIs(t, err, appErrors.ErrStoreNotFound)
	require.True(t, repository.findByIDCalled)
	require.False(t, repository.existsBySlugCalled)
	require.False(t, repository.updateCalled)
}

func TestUpdateStoreUseCaseExecute_ReturnsNotFoundWhenStoreDoesNotExist(t *testing.T) {
	repository := &fakeUpdateStoreRepository{}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := authenticatedContext(uuid.New())

	_, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     uuid.New(),
		Name:        "Updated Store",
		Slug:        "updated-store",
		Description: "Updated description",
	})

	require.ErrorIs(t, err, appErrors.ErrStoreNotFound)
	require.True(t, repository.findByIDCalled)
	require.False(t, repository.existsBySlugCalled)
	require.False(t, repository.updateCalled)
}

func TestUpdateStoreUseCaseExecute_PropagatesFindByIDError(t *testing.T) {
	expectedErr := errors.New("database unavailable")

	repository := &fakeUpdateStoreRepository{
		findByIDErr: expectedErr,
	}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := authenticatedContext(uuid.New())

	_, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     uuid.New(),
		Name:        "Updated Store",
		Slug:        "updated-store",
		Description: "Updated description",
	})

	require.ErrorIs(t, err, expectedErr)
	require.False(t, repository.existsBySlugCalled)
	require.False(t, repository.updateCalled)
}

func TestUpdateStoreUseCaseExecute_DoesNotCheckSlugWhenUnchanged(t *testing.T) {
	storeID := uuid.New()
	ownerID := uuid.New()

	store := mustReconstituteStore(t, storeID, ownerID)

	repository := &fakeUpdateStoreRepository{
		store: store,
	}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := authenticatedContext(ownerID)

	_, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     storeID,
		Name:        "Updated Store",
		Slug:        "test-store",
		Description: "Updated description",
	})

	require.NoError(t, err)
	require.False(t, repository.existsBySlugCalled)
	require.True(t, repository.updateCalled)
}

func TestUpdateStoreUseCaseExecute_RejectsExistingSlug(t *testing.T) {
	storeID := uuid.New()
	ownerID := uuid.New()

	store := mustReconstituteStore(t, storeID, ownerID)

	repository := &fakeUpdateStoreRepository{
		store:        store,
		existsBySlug: true,
	}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := authenticatedContext(ownerID)

	_, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     storeID,
		Name:        "Updated Store",
		Slug:        "another-store",
		Description: "Updated description",
	})

	require.ErrorIs(t, err, appErrors.ErrStoreSlugAlreadyExists)
	require.True(t, repository.existsBySlugCalled)
	require.Equal(t, "another-store", repository.lastExistsSlug)
	require.False(t, repository.updateCalled)
}

func TestUpdateStoreUseCaseExecute_PropagatesSlugLookupError(t *testing.T) {
	expectedErr := errors.New("slug lookup failed")

	storeID := uuid.New()
	ownerID := uuid.New()

	store := mustReconstituteStore(t, storeID, ownerID)

	repository := &fakeUpdateStoreRepository{
		store:           store,
		existsBySlugErr: expectedErr,
	}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := authenticatedContext(ownerID)

	_, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     storeID,
		Name:        "Updated Store",
		Slug:        "another-store",
		Description: "Updated description",
	})

	require.ErrorIs(t, err, expectedErr)
	require.False(t, repository.updateCalled)
}

func TestUpdateStoreUseCaseExecute_PropagatesUpdateError(t *testing.T) {
	expectedErr := errors.New("update failed")

	storeID := uuid.New()
	ownerID := uuid.New()

	store := mustReconstituteStore(t, storeID, ownerID)

	repository := &fakeUpdateStoreRepository{
		store:     store,
		updateErr: expectedErr,
	}

	useCase := use_cases.NewUpdateStoreUseCase(
		repository,
		&fakeUpdateStoreLogger{},
		&fakeUpdateStoreMetrics{},
	)

	ctx := authenticatedContext(ownerID)

	_, err := useCase.Execute(ctx, dto.UpdateStoreInput{
		StoreID:     storeID,
		Name:        "Updated Store",
		Slug:        "updated-store",
		Description: "Updated description",
	})

	require.ErrorIs(t, err, expectedErr)
	require.True(t, repository.updateCalled)
}

func mustReconstituteStore(
	t *testing.T,
	storeID uuid.UUID,
	ownerID uuid.UUID,
) *entities.Store {
	t.Helper()

	store, err := entities.ReconstituteStore(
		storeID,
		ownerID,
		"Test Store",
		"test-store",
		"Test store description",
		nil,
		string(valueobjects.PlanTypeBasic),
		string(valueobjects.StatusActive),
		testTimestamp(),
		testTimestamp(),
	)

	require.NoError(t, err)

	return store
}
