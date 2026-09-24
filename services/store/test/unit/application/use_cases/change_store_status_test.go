package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	use_cases "github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// TestChangeStoreStatusUseCase_ActivatesStore verifies that a trusted internal
// status request can activate an inactive Store.
//
// Authentication and subscription verification are intentionally absent from
// this application-level workflow. Those concerns belong to the presentation
// or service-to-service boundary before this use case is invoked.
func TestChangeStoreStatusUseCase_ActivatesStore(t *testing.T) {
	t.Parallel()

	store := newChangeStoreStatusTestStore(t, valueobjects.StatusInactive)

	repository := &fakeChangeStoreStatusRepository{
		store: store,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	output, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: store.ID().Value(),
			Status:  string(valueobjects.StatusActive),
		},
	)

	require.NoError(t, err)

	assert.Equal(t, valueobjects.StatusActive, output.Status)
	assert.Equal(t, valueobjects.StatusActive, store.Status())

	assert.True(t, repository.findByIDCalled)
	assert.True(t, repository.changeStatusCalled)
	assert.Equal(t, store.ID().Value(), repository.changeStatusStoreID)
	assert.Equal(t, string(valueobjects.StatusActive), repository.changeStatusValue)
}

// TestChangeStoreStatusUseCase_DeactivatesStore verifies that a trusted
// internal status request can deactivate an active Store.
func TestChangeStoreStatusUseCase_DeactivatesStore(t *testing.T) {
	t.Parallel()

	store := newChangeStoreStatusTestStore(t, valueobjects.StatusActive)

	repository := &fakeChangeStoreStatusRepository{
		store: store,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	output, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: store.ID().Value(),
			Status:  string(valueobjects.StatusInactive),
		},
	)

	require.NoError(t, err)

	assert.Equal(t, valueobjects.StatusInactive, output.Status)
	assert.Equal(t, valueobjects.StatusInactive, store.Status())

	assert.True(t, repository.changeStatusCalled)
	assert.Equal(t, string(valueobjects.StatusInactive), repository.changeStatusValue)
}

// TestChangeStoreStatusUseCase_IsIdempotent verifies that requesting the
// Store's current status remains successful.
//
// Billing/Subscription may legitimately send the same lifecycle state more
// than once. The domain Activate/Deactivate methods are idempotent, so the
// application should persist the resulting status without treating this as
// an error.
func TestChangeStoreStatusUseCase_IsIdempotent(t *testing.T) {
	t.Parallel()

	store := newChangeStoreStatusTestStore(t, valueobjects.StatusActive)

	repository := &fakeChangeStoreStatusRepository{
		store: store,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	output, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: store.ID().Value(),
			Status:  string(valueobjects.StatusActive),
		},
	)

	require.NoError(t, err)

	assert.Equal(t, valueobjects.StatusActive, output.Status)
	assert.Equal(t, valueobjects.StatusActive, store.Status())
	assert.True(t, repository.changeStatusCalled)
	assert.Equal(t, string(valueobjects.StatusActive), repository.changeStatusValue)
}

// TestChangeStoreStatusUseCase_RejectsInvalidStatus verifies that arbitrary
// status strings cannot reach the Store aggregate or persistence boundary.
func TestChangeStoreStatusUseCase_RejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	store := newChangeStoreStatusTestStore(t, valueobjects.StatusActive)

	repository := &fakeChangeStoreStatusRepository{
		store: store,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: store.ID().Value(),
			Status:  "suspended",
		},
	)

	require.Error(t, err)

	// Invalid status must be rejected by the application before the aggregate
	// is mutated or the repository persists anything.
	assert.False(t, repository.changeStatusCalled)
	assert.Equal(t, valueobjects.StatusActive, store.Status())
}

// TestChangeStoreStatusUseCase_ReturnsNotFoundWhenRepositoryReturnsNil
// verifies that a missing Store is represented by the application-level
// not-found error.
func TestChangeStoreStatusUseCase_ReturnsNotFoundWhenRepositoryReturnsNil(t *testing.T) {
	t.Parallel()

	repository := &fakeChangeStoreStatusRepository{
		store: nil,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: uuid.New(),
			Status:  string(valueobjects.StatusActive),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)

	assert.True(t, repository.findByIDCalled)
	assert.False(t, repository.changeStatusCalled)
}

// TestChangeStoreStatusUseCase_PropagatesStoreLookupError verifies that a
// persistence lookup failure is not converted into a not-found result.
func TestChangeStoreStatusUseCase_PropagatesStoreLookupError(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("database unavailable")

	repository := &fakeChangeStoreStatusRepository{
		findByIDErr: lookupErr,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: uuid.New(),
			Status:  string(valueobjects.StatusActive),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
	assert.False(t, repository.changeStatusCalled)
}

// TestChangeStoreStatusUseCase_PropagatesPersistenceError verifies that a
// repository failure remains visible to the caller.
func TestChangeStoreStatusUseCase_PropagatesPersistenceError(t *testing.T) {
	t.Parallel()

	store := newChangeStoreStatusTestStore(t, valueobjects.StatusInactive)
	persistenceErr := errors.New("status persistence failed")

	repository := &fakeChangeStoreStatusRepository{
		store:           store,
		changeStatusErr: persistenceErr,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: store.ID().Value(),
			Status:  string(valueobjects.StatusActive),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, persistenceErr)

	assert.True(t, repository.changeStatusCalled)

	// The aggregate was changed before persistence failed. The use case must
	// report the persistence failure rather than pretending the operation
	// succeeded.
	assert.Equal(t, valueobjects.StatusActive, store.Status())
}

// TestChangeStoreStatusUseCase_DoesNotRequireAuthenticatedIdentity verifies
// that authentication is not an application concern for this internal
// Billing/Subscription-triggered operation.
//
// The trusted presentation/service boundary is responsible for ensuring that
// the caller is authorized before invoking this use case.
func TestChangeStoreStatusUseCase_DoesNotRequireAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	store := newChangeStoreStatusTestStore(t, valueobjects.StatusInactive)

	repository := &fakeChangeStoreStatusRepository{
		store: store,
	}

	uc := use_cases.NewChangeStoreStatusUseCase(
		repository,
		&fakeChangeStoreStatusLogger{},
		&fakeChangeStoreStatusMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChangeStoreStatusInput{
			StoreID: store.ID().Value(),
			Status:  string(valueobjects.StatusActive),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, valueobjects.StatusActive, store.Status())
	assert.True(t, repository.changeStatusCalled)
}

// newChangeStoreStatusTestStore creates a valid Store aggregate in the
// requested initial status.
func newChangeStoreStatusTestStore(
	t *testing.T,
	status valueobjects.Status,
) *entities.Store {
	t.Helper()

	store, err := entities.NewStore(
		uuid.New(),
		"Test Store",
		"test-store",
		"Test store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)

	require.NoError(t, err)
	require.NotNil(t, store)

	switch status {
	case valueobjects.StatusActive:
		// NewStore creates active Stores by contract.

	case valueobjects.StatusInactive:
		require.NoError(t, store.Deactivate())

	default:
		t.Fatalf("unsupported test status: %s", status)
	}

	return store
}

// fakeChangeStoreStatusRepository implements the complete StoreRepository
// port while exposing only the behavior required by these tests.
type fakeChangeStoreStatusRepository struct {
	store *entities.Store

	findByIDCalled bool
	findByIDErr    error

	changeStatusCalled  bool
	changeStatusStoreID uuid.UUID
	changeStatusValue   string
	changeStatusErr     error
}

func (f *fakeChangeStoreStatusRepository) Create(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *fakeChangeStoreStatusRepository) Delete(
	_ context.Context,
	_ uuid.UUID,
) error {
	return nil
}

func (f *fakeChangeStoreStatusRepository) FindByID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Store, error) {
	f.findByIDCalled = true

	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}

	return f.store, nil
}

func (f *fakeChangeStoreStatusRepository) FindByOwnerID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeChangeStoreStatusRepository) FindBySlug(
	_ context.Context,
	_ string,
) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeChangeStoreStatusRepository) ExistsBySlug(
	_ context.Context,
	_ string,
) (bool, error) {
	return false, nil
}

func (f *fakeChangeStoreStatusRepository) ListActive(
	_ context.Context,
	_ string,
	_ int,
	_ int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *fakeChangeStoreStatusRepository) Update(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *fakeChangeStoreStatusRepository) ChangePlan(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *fakeChangeStoreStatusRepository) ChangeStatus(
	_ context.Context,
	storeID uuid.UUID,
	status string,
) error {
	f.changeStatusCalled = true
	f.changeStatusStoreID = storeID
	f.changeStatusValue = status

	return f.changeStatusErr
}

type fakeChangeStoreStatusLogger struct {
	events []ports.LogEvent
}

func (f *fakeChangeStoreStatusLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)
	return nil
}

type fakeChangeStoreStatusMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *fakeChangeStoreStatusMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return nil
}

func (f *fakeChangeStoreStatusMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return nil
}
