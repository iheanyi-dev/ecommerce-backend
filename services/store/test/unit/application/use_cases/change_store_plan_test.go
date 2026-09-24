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
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// TestChangeStorePlanUseCase_ChangesPlanToPremiumAndDeactivatesStore verifies
// the complete Premium plan transition.
//
// The application workflow is:
//
//	authenticated owner
//	    -> load Store
//	    -> verify ownership
//	    -> construct target Plan
//	    -> obtain current product count
//	    -> verify target Plan can accommodate products
//	    -> change the aggregate plan
//	    -> deactivate when moving to Premium
//	    -> persist through ChangePlan
func TestChangeStorePlanUseCase_ChangesPlanToPremiumAndDeactivatesStore(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	productCountProvider := &fakeChangeStorePlanProductCountProvider{
		count: 1,
	}

	logger := &fakeChangeStorePlanLogger{}
	metrics := &fakeChangeStorePlanMetrics{}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		productCountProvider,
		logger,
		metrics,
	)

	ctx := authenticatedChangeStorePlanContext(ownerID)

	output, err := uc.Execute(
		ctx,
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.NoError(t, err)

	require.NotNil(t, output.Plan)
	assert.Equal(t, valueobjects.PlanTypePremium, output.Plan.Type())

	// Moving to Premium is deliberately coupled with deactivation in this
	// workflow. Subscription/Billing reactivation is a separate concern.
	assert.Equal(t, valueobjects.StatusInactive, output.Status)

	require.NotNil(t, repository.changedPlanStore)
	assert.Same(t, store, repository.changedPlanStore)

	assert.True(t, repository.changePlanCalled)
	assert.Equal(t, store.ID().Value(), repository.changePlanStoreID)

	assert.True(t, productCountProvider.called)
	assert.Equal(t, store.ID().Value(), productCountProvider.storeID)
}

// TestChangeStorePlanUseCase_ChangesPlanToBasicWithoutDeactivating verifies
// that changing to Basic does not deactivate the Store.
func TestChangeStorePlanUseCase_ChangesPlanToBasicWithoutDeactivating(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	productCountProvider := &fakeChangeStorePlanProductCountProvider{
		count: 2,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		productCountProvider,
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	ctx := authenticatedChangeStorePlanContext(ownerID)

	output, err := uc.Execute(
		ctx,
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypeBasic),
		},
	)

	require.NoError(t, err)

	require.NotNil(t, output.Plan)
	assert.Equal(t, valueobjects.PlanTypeBasic, output.Plan.Type())
	assert.Equal(t, valueobjects.StatusActive, output.Status)

	assert.True(t, repository.changePlanCalled)
	assert.Same(t, store, repository.changedPlanStore)
}

// TestChangeStorePlanUseCase_RequiresAuthentication verifies that Store plan
// changes cannot be initiated without a trusted authenticated identity.
func TestChangeStorePlanUseCase_RequiresAuthentication(t *testing.T) {
	t.Parallel()

	repository := &fakeChangeStorePlanRepository{}
	productCountProvider := &fakeChangeStorePlanProductCountProvider{}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		productCountProvider,
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		context.Background(),
		dto.ChangeStorePlanInput{
			StoreID:  uuid.New(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, applicationerrors.ErrUnauthenticated)

	assert.False(t, repository.findByIDCalled)
	assert.False(t, repository.changePlanCalled)
	assert.False(t, productCountProvider.called)
}

// TestChangeStorePlanUseCase_HidesNonOwnedStore verifies the owner privacy
// rule used by the other owner-only Store workflows.
func TestChangeStorePlanUseCase_HidesNonOwnedStore(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	callerID := uuid.New()

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	productCountProvider := &fakeChangeStorePlanProductCountProvider{}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		productCountProvider,
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(callerID),
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)

	assert.True(t, repository.findByIDCalled)
	assert.False(t, productCountProvider.called)
	assert.False(t, repository.changePlanCalled)
}

// TestChangeStorePlanUseCase_ReturnsNotFoundWhenRepositoryReturnsNil verifies
// that a missing Store is represented by the application-level not-found
// error.
func TestChangeStorePlanUseCase_ReturnsNotFoundWhenRepositoryReturnsNil(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()

	repository := &fakeChangeStorePlanRepository{
		store: nil,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		&fakeChangeStorePlanProductCountProvider{},
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(ownerID),
		dto.ChangeStorePlanInput{
			StoreID:  uuid.New(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)

	assert.True(t, repository.findByIDCalled)
	assert.False(t, repository.changePlanCalled)
}

// TestChangeStorePlanUseCase_PropagatesStoreLookupError verifies that a
// repository lookup failure is not converted into a not-found result.
func TestChangeStorePlanUseCase_PropagatesStoreLookupError(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	lookupErr := errors.New("database unavailable")

	repository := &fakeChangeStorePlanRepository{
		findByIDErr: lookupErr,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		&fakeChangeStorePlanProductCountProvider{},
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(ownerID),
		dto.ChangeStorePlanInput{
			StoreID:  uuid.New(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, lookupErr)
	assert.False(t, repository.changePlanCalled)
}

// TestChangeStorePlanUseCase_RejectsInvalidTargetPlan verifies that arbitrary
// plan strings are rejected by the domain Plan factory.
func TestChangeStorePlanUseCase_RejectsInvalidTargetPlan(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	productCountProvider := &fakeChangeStorePlanProductCountProvider{
		count: 1,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		productCountProvider,
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(ownerID),
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: "enterprise",
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, domainerrors.ErrInvalidPlanType)

	assert.False(t, productCountProvider.called)
	assert.False(t, repository.changePlanCalled)
}

// TestChangeStorePlanUseCase_RejectsPlanWhenProductCountExceedsTargetLimit
// verifies that product-capacity validation occurs before mutating the Store
// aggregate.
func TestChangeStorePlanUseCase_RejectsPlanWhenProductCountExceedsTargetLimit(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypePremium,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	productCountProvider := &fakeChangeStorePlanProductCountProvider{
		count: 3, // Basic permits only 2 products.
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		productCountProvider,
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(ownerID),
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypeBasic),
		},
	)

	require.Error(t, err)

	// The exact capacity error will be established by the production
	// application contract rather than introducing a duplicate domain rule.
	assert.Equal(t, valueobjects.PlanTypePremium, store.Plan().Type())
	assert.Equal(t, valueobjects.StatusActive, store.Status())

	assert.True(t, productCountProvider.called)
	assert.False(t, repository.changePlanCalled)
}

// TestChangeStorePlanUseCase_PropagatesProductCountError verifies that an
// inability to determine the current product count prevents plan mutation.
func TestChangeStorePlanUseCase_PropagatesProductCountError(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	countErr := errors.New("product service unavailable")

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	productCountProvider := &fakeChangeStorePlanProductCountProvider{
		countErr: countErr,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		productCountProvider,
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(ownerID),
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, countErr)

	assert.Equal(t, valueobjects.PlanTypeBasic, store.Plan().Type())
	assert.Equal(t, valueobjects.StatusActive, store.Status())
	assert.False(t, repository.changePlanCalled)
}

// TestChangeStorePlanUseCase_PropagatesPersistenceError verifies that the
// repository error remains visible when persistence fails.
func TestChangeStorePlanUseCase_PropagatesPersistenceError(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	persistenceErr := errors.New("change plan persistence failed")

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store:         store,
		changePlanErr: persistenceErr,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		&fakeChangeStorePlanProductCountProvider{count: 1},
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(ownerID),
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, persistenceErr)

	assert.True(t, repository.changePlanCalled)
	assert.Same(t, store, repository.changedPlanStore)

	// The repository failed after the aggregate had been changed in memory.
	// The application must not pretend persistence succeeded.
	assert.Equal(t, valueobjects.PlanTypePremium, store.Plan().Type())
	assert.Equal(t, valueobjects.StatusInactive, store.Status())
}

// TestChangeStorePlanUseCase_SamePlanIsIdempotent verifies that requesting
// the plan the Store already has does not produce a domain error.
//
// Persistence still goes through ChangePlan so the application has one
// persistence boundary for plan-change requests.
func TestChangeStorePlanUseCase_SamePlanIsIdempotent(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		&fakeChangeStorePlanProductCountProvider{count: 1},
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(ownerID),
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypeBasic),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, valueobjects.PlanTypeBasic, store.Plan().Type())
	assert.Equal(t, valueobjects.StatusActive, store.Status())

	assert.True(t, repository.changePlanCalled)
	assert.Same(t, store, repository.changedPlanStore)
}

// TestChangeStorePlanUseCase_UsesAuthenticatedOwnerNotInputOwner verifies
// that ownership comes exclusively from the trusted authentication context.
func TestChangeStorePlanUseCase_UsesAuthenticatedOwnerNotInputOwner(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	callerID := uuid.New()

	store := newTestStore(
		t,
		ownerID,
		valueobjects.PlanTypeBasic,
		valueobjects.StatusActive,
	)

	repository := &fakeChangeStorePlanRepository{
		store: store,
	}

	uc := use_cases.NewChangeStorePlanUseCase(
		repository,
		&fakeChangeStorePlanProductCountProvider{count: 1},
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)

	_, err := uc.Execute(
		authenticatedChangeStorePlanContext(callerID),
		dto.ChangeStorePlanInput{
			StoreID:  store.ID().Value(),
			PlanType: string(valueobjects.PlanTypePremium),
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
	assert.False(t, repository.changePlanCalled)
}

// newTestStore creates a valid Store aggregate for application-layer tests.
func newTestStore(
	t *testing.T,
	ownerID uuid.UUID,
	planType valueobjects.PlanType,
	status valueobjects.Status,
) *entities.Store {
	t.Helper()

	plan, err := valueobjects.NewPlan(planType)
	require.NoError(t, err)
	require.NotNil(t, plan)

	store, err := entities.NewStore(
		ownerID,
		"Test Store",
		"test-store",
		"Test store description",
		nil,
		string(planType),
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

func authenticatedChangeStorePlanContext(userID uuid.UUID) context.Context {
	return ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{
			UserID: userID,
			Role:   "user",
		},
	)
}

// fakeChangeStorePlanRepository is local to this test because ChangePlan is
// the new repository capability being introduced by this workflow.
type fakeChangeStorePlanRepository struct {
	store *entities.Store

	findByIDCalled bool
	findByIDErr    error

	changePlanCalled  bool
	changePlanStoreID uuid.UUID
	changedPlanStore  *entities.Store
	changePlanErr     error
}

func (f *fakeChangeStorePlanRepository) Create(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *fakeChangeStorePlanRepository) Delete(
	_ context.Context,
	_ uuid.UUID,
) error {
	return nil
}

func (f *fakeChangeStorePlanRepository) FindByID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Store, error) {
	f.findByIDCalled = true

	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}

	return f.store, nil
}

func (f *fakeChangeStorePlanRepository) FindByOwnerID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeChangeStorePlanRepository) FindBySlug(
	_ context.Context,
	_ string,
) (*entities.Store, error) {
	return nil, nil
}

func (f *fakeChangeStorePlanRepository) ExistsBySlug(
	_ context.Context,
	_ string,
) (bool, error) {
	return false, nil
}

func (f *fakeChangeStorePlanRepository) ListActive(
	_ context.Context,
	_ string,
	_ int,
	_ int,
) ([]*entities.Store, int, error) {
	return nil, 0, nil
}

func (f *fakeChangeStorePlanRepository) Update(
	_ context.Context,
	_ *entities.Store,
) error {
	return nil
}

func (f *fakeChangeStorePlanRepository) ChangePlan(
	_ context.Context,
	store *entities.Store,
) error {
	f.changePlanCalled = true
	f.changedPlanStore = store

	if store != nil {
		f.changePlanStoreID = store.ID().Value()
	}

	return f.changePlanErr
}

func (f *fakeChangeStorePlanRepository) ChangeStatus(
	_ context.Context,
	_ uuid.UUID,
	_ string,
) error {
	return nil
}

type fakeChangeStorePlanProductCountProvider struct {
	count    int
	countErr error

	called  bool
	storeID uuid.UUID
}

func (f *fakeChangeStorePlanProductCountProvider) CountByStoreID(
	_ context.Context,
	storeID uuid.UUID,
) (int, error) {
	f.called = true
	f.storeID = storeID

	if f.countErr != nil {
		return 0, f.countErr
	}

	return f.count, nil
}

type fakeChangeStorePlanLogger struct {
	events []ports.LogEvent
}

func (f *fakeChangeStorePlanLogger) Log(
	_ context.Context,
	event ports.LogEvent,
) error {
	f.events = append(f.events, event)
	return nil
}

type fakeChangeStorePlanMetrics struct {
	incrementCalls int
	observeCalls   int
}

func (f *fakeChangeStorePlanMetrics) Increment(
	_ context.Context,
	_ ports.Metric,
) error {
	f.incrementCalls++
	return nil
}

func (f *fakeChangeStorePlanMetrics) Observe(
	_ context.Context,
	_ ports.Metric,
) error {
	f.observeCalls++
	return nil
}
