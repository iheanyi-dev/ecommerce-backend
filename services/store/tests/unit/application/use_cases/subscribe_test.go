package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

func TestSubscribeUseCase_Success(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 2)
	planID := uuid.New()

	repository := &subscribeStoreRepositoryFake{store: store}
	identity := &subscribeIdentityProviderFake{
		user: &ports.IdentityUser{ID: ownerID, Email: " owner@example.com "},
	}
	billing := &subscribeBillingProviderFake{
		plan: &ports.BillingPlan{
			ID: planID, Type: string(valueobjects.PlanTypePremium), Available: true,
		},
		checkout: &ports.BillingSubscribeOutput{
			SubscriptionID: uuid.New(),
			PaymentID:      uuid.New(),
			CheckoutURL:    "https://checkout.example.com/session",
		},
	}
	logger := &subscribeLoggerFake{}
	metrics := &subscribeMetricsFake{}

	uc := use_cases.NewSubscribeUseCase(repository, identity, billing, logger, metrics)

	output, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.SubscribeInput{
		StoreID: store.ID().Value(),
		PlanID:  planID,
	})

	require.NoError(t, err)
	assert.Equal(t, billing.checkout.SubscriptionID, output.SubscriptionID)
	assert.Equal(t, billing.checkout.PaymentID, output.PaymentID)
	assert.Equal(t, billing.checkout.CheckoutURL, output.CheckoutURL)

	assert.True(t, repository.findByIDCalled)
	assert.True(t, identity.getUserCalled)
	assert.True(t, billing.getPlanCalled)
	assert.True(t, billing.subscribeCalled)

	assert.Equal(t, ownerID, identity.requestedUserID)
	assert.Equal(t, "owner@example.com", billing.subscribeInput.CustomerEmail)
	assert.Equal(t, store.ID().Value(), billing.subscribeInput.StoreID)
	assert.Equal(t, ownerID, billing.subscribeInput.PayerID)
	assert.Equal(t, planID, billing.subscribeInput.PlanID)

	// Starting checkout must not activate or change the Store's plan.
	assert.Equal(t, valueobjects.PlanTypeBasic, store.Plan().Type())
	assert.False(t, repository.changePlanCalled)
	assert.False(t, repository.changeStatusCalled)

	assert.NotEmpty(t, logger.events)
	assert.Equal(t, 1, metrics.incrementCalls)
	assert.Equal(t, 1, metrics.observeCalls)
}

func TestSubscribeUseCase_RejectsUnauthenticatedRequest(t *testing.T) {
	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{},
		&subscribeIdentityProviderFake{},
		&subscribeBillingProviderFake{},
	)

	_, err := uc.Execute(context.Background(), dto.SubscribeInput{
		StoreID: uuid.New(), PlanID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrUnauthenticated)
}

func TestSubscribeUseCase_RejectsInvalidRequest(t *testing.T) {
	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{},
		&subscribeIdentityProviderFake{},
		&subscribeBillingProviderFake{},
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(uuid.New()), dto.SubscribeInput{
		StoreID: uuid.Nil, PlanID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrInvalidSubscriptionRequest)
}

func TestSubscribeUseCase_RejectsMissingStore(t *testing.T) {
	repository := &subscribeStoreRepositoryFake{}
	uc := newSubscribeTestUseCase(
		repository, &subscribeIdentityProviderFake{}, &subscribeBillingProviderFake{},
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(uuid.New()), dto.SubscribeInput{
		StoreID: uuid.New(), PlanID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrStoreNotFound)
}

func TestSubscribeUseCase_HidesStoreFromNonOwner(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	billing := &subscribeBillingProviderFake{}

	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{store: store},
		&subscribeIdentityProviderFake{},
		billing,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(uuid.New()), dto.SubscribeInput{
		StoreID: store.ID().Value(), PlanID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrStoreNotFound)
	assert.False(t, billing.getPlanCalled)
}

func TestSubscribeUseCase_RejectsUnavailablePlan(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	planID := uuid.New()

	billing := &subscribeBillingProviderFake{
		plan: &ports.BillingPlan{
			ID: planID, Type: string(valueobjects.PlanTypePremium), Available: false,
		},
	}
	identity := &subscribeIdentityProviderFake{
		user: &ports.IdentityUser{ID: ownerID, Email: "owner@example.com"},
	}

	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{store: store}, identity, billing,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.SubscribeInput{
		StoreID: store.ID().Value(), PlanID: planID,
	})

	assert.ErrorIs(t, err, apperrors.ErrBillingPlanNotAvailable)
	assert.False(t, identity.getUserCalled)
	assert.False(t, billing.subscribeCalled)
}

func TestSubscribeUseCase_RejectsUnsupportedPlan(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	planID := uuid.New()

	billing := &subscribeBillingProviderFake{
		plan: &ports.BillingPlan{ID: planID, Type: "enterprise", Available: true},
	}
	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{store: store},
		&subscribeIdentityProviderFake{},
		billing,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.SubscribeInput{
		StoreID: store.ID().Value(), PlanID: planID,
	})

	assert.ErrorIs(t, err, apperrors.ErrUnsupportedStorePlan)
}

func TestSubscribeUseCase_RejectsPlanThatCannotAccommodateProducts(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 3)
	planID := uuid.New()

	billing := &subscribeBillingProviderFake{
		plan: &ports.BillingPlan{
			ID: planID, Type: string(valueobjects.PlanTypeBasic), Available: true,
		},
	}
	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{store: store},
		&subscribeIdentityProviderFake{},
		billing,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.SubscribeInput{
		StoreID: store.ID().Value(), PlanID: planID,
	})

	assert.ErrorIs(t, err, apperrors.ErrStorePlanProductLimitExceeded)
	assert.False(t, billing.subscribeCalled)
}

func TestSubscribeUseCase_RejectsUnavailableOwnerEmail(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	planID := uuid.New()

	billing := &subscribeBillingProviderFake{
		plan: &ports.BillingPlan{
			ID: planID, Type: string(valueobjects.PlanTypePremium), Available: true,
		},
	}
	identity := &subscribeIdentityProviderFake{
		user: &ports.IdentityUser{ID: ownerID, Email: " "},
	}

	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{store: store}, identity, billing,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.SubscribeInput{
		StoreID: store.ID().Value(), PlanID: planID,
	})

	assert.ErrorIs(t, err, apperrors.ErrIdentityUserUnavailable)
	assert.False(t, billing.subscribeCalled)
}

func TestSubscribeUseCase_PropagatesBillingSubscribeFailure(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	planID := uuid.New()
	billingErr := errors.New("Billing unavailable")

	billing := &subscribeBillingProviderFake{
		plan: &ports.BillingPlan{
			ID: planID, Type: string(valueobjects.PlanTypePremium), Available: true,
		},
		subscribeErr: billingErr,
	}
	identity := &subscribeIdentityProviderFake{
		user: &ports.IdentityUser{ID: ownerID, Email: "owner@example.com"},
	}

	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{store: store}, identity, billing,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.SubscribeInput{
		StoreID: store.ID().Value(), PlanID: planID,
	})

	assert.ErrorIs(t, err, billingErr)
}

func TestSubscribeUseCase_RejectsMissingCheckoutURL(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	planID := uuid.New()

	billing := &subscribeBillingProviderFake{
		plan: &ports.BillingPlan{
			ID: planID, Type: string(valueobjects.PlanTypePremium), Available: true,
		},
		checkout: &ports.BillingSubscribeOutput{
			SubscriptionID: uuid.New(),
			PaymentID:      uuid.New(),
		},
	}
	identity := &subscribeIdentityProviderFake{
		user: &ports.IdentityUser{ID: ownerID, Email: "owner@example.com"},
	}

	uc := newSubscribeTestUseCase(
		&subscribeStoreRepositoryFake{store: store}, identity, billing,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.SubscribeInput{
		StoreID: store.ID().Value(), PlanID: planID,
	})

	assert.ErrorIs(t, err, apperrors.ErrBillingCheckoutUnavailable)
}

func newSubscribeTestUseCase(
	repository *subscribeStoreRepositoryFake,
	identity *subscribeIdentityProviderFake,
	billing *subscribeBillingProviderFake,
) *use_cases.SubscribeUseCase {
	return use_cases.NewSubscribeUseCase(
		repository, identity, billing, &subscribeLoggerFake{}, &subscribeMetricsFake{},
	)
}

func subscribeTestStore(
	t *testing.T,
	ownerID uuid.UUID,
	planType valueobjects.PlanType,
	productCount int,
) *entities.Store {
	t.Helper()

	store, err := entities.NewStore(
		ownerID, "Test Store", "test-store", "Test store description", nil, string(planType),
	)
	require.NoError(t, err)

	store, err = entities.ReconstituteStoreWithProductCount(
		store.ID().Value(),
		ownerID,
		"Test Store",
		"test-store",
		"Test store description",
		nil,
		string(planType),
		string(valueobjects.StatusActive),
		productCount,
		store.CreatedAt(),
		store.UpdatedAt(),
	)
	require.NoError(t, err)

	return store
}

func subscribeAuthenticatedContext(userID uuid.UUID) context.Context {
	return ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{UserID: userID, Role: "user"},
	)
}

type subscribeStoreRepositoryFake struct {
	store              *entities.Store
	findByIDCalled     bool
	changePlanCalled   bool
	changeStatusCalled bool
}

func (f *subscribeStoreRepositoryFake) Create(context.Context, *entities.Store) error {
	return nil
}
func (f *subscribeStoreRepositoryFake) Delete(context.Context, uuid.UUID) error {
	return nil
}
func (f *subscribeStoreRepositoryFake) FindByID(context.Context, uuid.UUID) (*entities.Store, error) {
	f.findByIDCalled = true
	return f.store, nil
}
func (f *subscribeStoreRepositoryFake) FindByOwnerID(context.Context, uuid.UUID) (*entities.Store, error) {
	return nil, nil
}
func (f *subscribeStoreRepositoryFake) FindBySlug(context.Context, string) (*entities.Store, error) {
	return nil, nil
}
func (f *subscribeStoreRepositoryFake) ExistsBySlug(context.Context, string) (bool, error) {
	return false, nil
}
func (f *subscribeStoreRepositoryFake) ListActive(context.Context, string, int, int) ([]*entities.Store, int, error) {
	return nil, 0, nil
}
func (f *subscribeStoreRepositoryFake) Update(context.Context, *entities.Store) error {
	return nil
}
func (f *subscribeStoreRepositoryFake) ChangePlan(context.Context, *entities.Store) error {
	f.changePlanCalled = true
	return nil
}
func (f *subscribeStoreRepositoryFake) ChangeStatus(context.Context, uuid.UUID, string) error {
	f.changeStatusCalled = true
	return nil
}

type subscribeIdentityProviderFake struct {
	user            *ports.IdentityUser
	getUserCalled   bool
	requestedUserID uuid.UUID
	getUserErr      error
}

func (f *subscribeIdentityProviderFake) GetUser(
	_ context.Context,
	userID uuid.UUID,
) (*ports.IdentityUser, error) {
	f.getUserCalled = true
	f.requestedUserID = userID
	return f.user, f.getUserErr
}
func (*subscribeIdentityProviderFake) IsVendor(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}
func (*subscribeIdentityProviderFake) PromoteToVendor(context.Context, uuid.UUID) error {
	return nil
}

type subscribeBillingProviderFake struct {
	plan            *ports.BillingPlan
	getPlanCalled   bool
	getPlanErr      error
	subscribeCalled bool
	subscribeInput  ports.BillingSubscribeInput
	checkout        *ports.BillingSubscribeOutput
	subscribeErr    error
}

func (f *subscribeBillingProviderFake) GetPlan(
	context.Context,
	uuid.UUID,
) (*ports.BillingPlan, error) {
	f.getPlanCalled = true
	return f.plan, f.getPlanErr
}
func (f *subscribeBillingProviderFake) ListPlans(context.Context) ([]ports.BillingPlan, error) {
	return nil, nil
}
func (f *subscribeBillingProviderFake) GetSubscription(context.Context, uuid.UUID) (*ports.BillingSubscription, error) {
	return nil, nil
}
func (f *subscribeBillingProviderFake) ListStoreSubscriptions(context.Context, uuid.UUID) ([]ports.BillingSubscription, error) {
	return nil, nil
}

func (f *subscribeBillingProviderFake) Subscribe(
	_ context.Context,
	input ports.BillingSubscribeInput,
) (*ports.BillingSubscribeOutput, error) {
	f.subscribeCalled = true
	f.subscribeInput = input
	return f.checkout, f.subscribeErr
}

type subscribeLoggerFake struct {
	events []ports.LogEvent
}

func (f *subscribeLoggerFake) Log(_ context.Context, event ports.LogEvent) error {
	f.events = append(f.events, event)
	return nil
}

type subscribeMetricsFake struct {
	incrementCalls int
	observeCalls   int
}

func (f *subscribeMetricsFake) Increment(context.Context, ports.Metric) error {
	f.incrementCalls++
	return nil
}
func (f *subscribeMetricsFake) Observe(context.Context, ports.Metric) error {
	f.observeCalls++
	return nil
}
