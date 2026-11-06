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

func TestUpdateAfterSubscriptionUseCase_ActivatesPremiumStore(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypeBasic, valueobjects.StatusInactive,
	)
	repository := &fakeChangeStorePlanRepository{store: store}
	uc := newSubscriptionUpdateUseCase(repository)

	output, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "active", string(valueobjects.PlanTypePremium),
	))

	require.NoError(t, err)
	require.NotNil(t, output.Plan)
	assert.Equal(t, valueobjects.PlanTypePremium, output.Plan.Type())
	assert.Equal(t, valueobjects.StatusActive, output.Status)
	assert.True(t, repository.changePlanCalled)
	assert.Same(t, store, repository.changedPlanStore)
}

func TestUpdateAfterSubscriptionUseCase_ActivatesBasicStore(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypePremium, valueobjects.StatusInactive,
	)
	repository := &fakeChangeStorePlanRepository{store: store}
	uc := newSubscriptionUpdateUseCase(repository)

	output, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "active", string(valueobjects.PlanTypeBasic),
	))

	require.NoError(t, err)
	assert.Equal(t, valueobjects.PlanTypeBasic, output.Plan.Type())
	assert.Equal(t, valueobjects.StatusActive, output.Status)
	assert.True(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_DeactivatesExpiredStore(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypePremium, valueobjects.StatusActive,
	)
	repository := &fakeChangeStorePlanRepository{store: store}
	uc := newSubscriptionUpdateUseCase(repository)

	output, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "expired", "",
	))

	require.NoError(t, err)
	assert.Equal(t, valueobjects.PlanTypePremium, output.Plan.Type())
	assert.Equal(t, valueobjects.StatusInactive, output.Status)
	assert.True(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_DeactivatesCancelledStore(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypePremium, valueobjects.StatusActive,
	)
	repository := &fakeChangeStorePlanRepository{store: store}
	uc := newSubscriptionUpdateUseCase(repository)

	output, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "cancelled", "",
	))

	require.NoError(t, err)
	assert.Equal(t, valueobjects.StatusInactive, output.Status)
	assert.True(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_PendingDoesNotPersist(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypeBasic, valueobjects.StatusActive,
	)
	repository := &fakeChangeStorePlanRepository{store: store}
	uc := newSubscriptionUpdateUseCase(repository)

	output, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "pending", "",
	))

	require.NoError(t, err)
	assert.Equal(t, valueobjects.PlanTypeBasic, output.Plan.Type())
	assert.Equal(t, valueobjects.StatusActive, output.Status)
	assert.False(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_FailedDoesNotPersist(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypePremium, valueobjects.StatusActive,
	)
	repository := &fakeChangeStorePlanRepository{store: store}
	uc := newSubscriptionUpdateUseCase(repository)

	output, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "failed", "",
	))

	require.NoError(t, err)
	assert.Equal(t, valueobjects.StatusActive, output.Status)
	assert.False(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_RejectsUnknownStatus(t *testing.T) {
	t.Parallel()

	repository := &fakeChangeStorePlanRepository{}
	uc := newSubscriptionUpdateUseCase(repository)

	_, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		uuid.New(), "unknown", "",
	))

	require.ErrorIs(t, err, use_cases.ErrInvalidSubscriptionStatus)
	assert.False(t, repository.findByIDCalled)
	assert.False(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_RejectsInvalidPlan(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypeBasic, valueobjects.StatusInactive,
	)
	repository := &fakeChangeStorePlanRepository{store: store}
	uc := newSubscriptionUpdateUseCase(repository)

	_, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "active", "enterprise",
	))

	require.Error(t, err)
	assert.False(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	repository := &fakeChangeStorePlanRepository{}
	uc := newSubscriptionUpdateUseCase(repository)

	_, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		uuid.New(), "active", string(valueobjects.PlanTypePremium),
	))

	require.ErrorIs(t, err, applicationerrors.ErrStoreNotFound)
	assert.False(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_PropagatesLookupError(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("database unavailable")
	repository := &fakeChangeStorePlanRepository{findByIDErr: lookupErr}
	uc := newSubscriptionUpdateUseCase(repository)

	_, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		uuid.New(), "active", string(valueobjects.PlanTypePremium),
	))

	require.ErrorIs(t, err, lookupErr)
	assert.False(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_PropagatesPersistenceError(t *testing.T) {
	t.Parallel()

	store := newTestStore(
		t, uuid.New(), valueobjects.PlanTypeBasic, valueobjects.StatusInactive,
	)
	persistenceErr := errors.New("database unavailable")
	repository := &fakeChangeStorePlanRepository{
		store:         store,
		changePlanErr: persistenceErr,
	}
	uc := newSubscriptionUpdateUseCase(repository)

	_, err := uc.Execute(context.Background(), subscriptionUpdateInput(
		store.ID().Value(), "active", string(valueobjects.PlanTypePremium),
	))

	require.ErrorIs(t, err, persistenceErr)
	assert.True(t, repository.changePlanCalled)
}

func TestUpdateAfterSubscriptionUseCase_RequiresEventIdentifiers(t *testing.T) {
	t.Parallel()

	repository := &fakeChangeStorePlanRepository{}
	uc := newSubscriptionUpdateUseCase(repository)

	input := subscriptionUpdateInput(
		uuid.New(), "active", string(valueobjects.PlanTypePremium),
	)
	input.EventID = uuid.Nil

	_, err := uc.Execute(context.Background(), input)

	require.Error(t, err)
	assert.False(t, repository.findByIDCalled)
}

func newSubscriptionUpdateUseCase(
	repository *fakeChangeStorePlanRepository,
) *use_cases.UpdateAfterSubscriptionUseCase {
	return use_cases.NewUpdateAfterSubscriptionUseCase(
		repository,
		&fakeChangeStorePlanLogger{},
		&fakeChangeStorePlanMetrics{},
	)
}

func subscriptionUpdateInput(
	storeID uuid.UUID,
	status string,
	planType string,
) dto.UpdateAfterSubscriptionInput {
	return dto.UpdateAfterSubscriptionInput{
		EventID:        uuid.New(),
		SubscriptionID: uuid.New(),
		StoreID:        storeID,
		Status:         status,
		PlanType:       planType,
	}
}

// Compile-time assertion keeps the test context import explicit and verifies
// the application port remains the expected use case boundary.
var _ ports.UpdateAfterSubscriptionService = (*use_cases.UpdateAfterSubscriptionUseCase)(nil)
var _ = entities.Store{}
