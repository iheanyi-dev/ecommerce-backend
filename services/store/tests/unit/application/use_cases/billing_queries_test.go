package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

type billingQueriesProviderFake struct {
	plans                []ports.BillingPlan
	plan                 *ports.BillingPlan
	subscriptions        []ports.BillingSubscription
	subscription         *ports.BillingSubscription
	listPlansErr         error
	getPlanErr           error
	listSubscriptionsErr error
	getSubscriptionErr   error

	listPlansCalled         bool
	getPlanCalled           bool
	listSubscriptionsCalled bool
	getSubscriptionCalled   bool

	requestedPlanID         uuid.UUID
	requestedStoreID        uuid.UUID
	requestedSubscriptionID uuid.UUID
}

func (f *billingQueriesProviderFake) GetPlan(
	_ context.Context,
	planID uuid.UUID,
) (*ports.BillingPlan, error) {
	f.getPlanCalled = true
	f.requestedPlanID = planID
	return f.plan, f.getPlanErr
}

func (f *billingQueriesProviderFake) ListPlans(
	context.Context,
) ([]ports.BillingPlan, error) {
	f.listPlansCalled = true
	return f.plans, f.listPlansErr
}

func (f *billingQueriesProviderFake) GetSubscription(
	_ context.Context,
	subscriptionID uuid.UUID,
) (*ports.BillingSubscription, error) {
	f.getSubscriptionCalled = true
	f.requestedSubscriptionID = subscriptionID
	return f.subscription, f.getSubscriptionErr
}

func (f *billingQueriesProviderFake) ListStoreSubscriptions(
	_ context.Context,
	storeID uuid.UUID,
) ([]ports.BillingSubscription, error) {
	f.listSubscriptionsCalled = true
	f.requestedStoreID = storeID
	return f.subscriptions, f.listSubscriptionsErr
}

func (f *billingQueriesProviderFake) Subscribe(
	context.Context,
	ports.BillingSubscribeInput,
) (*ports.BillingSubscribeOutput, error) {
	return nil, errors.New("unexpected Subscribe call")
}

func TestListBillingPlansUseCase_Success(t *testing.T) {
	startsAt := time.Now().UTC()
	endsAt := startsAt.Add(24 * time.Hour)
	createdAt := startsAt.Add(-time.Hour)

	provider := &billingQueriesProviderFake{
		plans: []ports.BillingPlan{
			{
				ID:          uuid.New(),
				Type:        string(valueobjects.PlanTypePremium),
				Name:        "Premium",
				Description: "Premium Store plan",
				Price: ports.BillingMoney{
					AmountMinor: 500000,
					Currency:    "NGN",
				},
				Duration: 30 * 24 * time.Hour,
				Capabilities: ports.BillingPlanCapabilities{
					MaxProducts: 5,
					AIEnabled:   true,
				},
				Available: true,
				Promotion: &ports.BillingPlanPromotion{
					Name:                "Launch offer",
					DiscountAmountMinor: 50000,
					StartsAt:            &startsAt,
					EndsAt:              &endsAt,
				},
				CreatedAt: &createdAt,
			},
		},
	}

	uc := use_cases.NewListBillingPlansUseCase(provider)

	output, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, output, 1)
	assert.True(t, provider.listPlansCalled)
	assert.Equal(t, provider.plans[0].ID, output[0].ID)
	assert.Equal(t, "Premium", output[0].Name)
	assert.Equal(t, int64(500000), output[0].Price.AmountMinor)
	assert.Equal(t, "NGN", output[0].Price.Currency)
	assert.Equal(t, 5, output[0].MaxProducts)
	assert.True(t, output[0].AIEnabled)
	require.NotNil(t, output[0].Promotion)
	assert.Equal(t, "Launch offer", output[0].Promotion.Name)
	assert.Equal(t, &startsAt, output[0].Promotion.StartsAt)
	assert.Equal(t, &createdAt, output[0].CreatedAt)
}

func TestListBillingPlansUseCase_EmptyCatalogue(t *testing.T) {
	provider := &billingQueriesProviderFake{}
	uc := use_cases.NewListBillingPlansUseCase(provider)

	output, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Empty(t, output)
	assert.True(t, provider.listPlansCalled)
}

func TestListBillingPlansUseCase_PropagatesProviderError(t *testing.T) {
	providerErr := errors.New("Billing unavailable")
	provider := &billingQueriesProviderFake{listPlansErr: providerErr}
	uc := use_cases.NewListBillingPlansUseCase(provider)

	_, err := uc.Execute(context.Background())

	assert.ErrorIs(t, err, providerErr)
}

func TestGetBillingPlanUseCase_Success(t *testing.T) {
	planID := uuid.New()
	provider := &billingQueriesProviderFake{
		plan: &ports.BillingPlan{
			ID:        planID,
			Type:      string(valueobjects.PlanTypePremium),
			Name:      "Premium",
			Available: true,
		},
	}
	uc := use_cases.NewGetBillingPlanUseCase(provider)

	output, err := uc.Execute(context.Background(), dto.GetBillingPlanInput{
		PlanID: planID,
	})

	require.NoError(t, err)
	assert.True(t, provider.getPlanCalled)
	assert.Equal(t, planID, provider.requestedPlanID)
	assert.Equal(t, planID, output.ID)
	assert.Equal(t, "Premium", output.Name)
}

func TestGetBillingPlanUseCase_RejectsNilPlanID(t *testing.T) {
	provider := &billingQueriesProviderFake{}
	uc := use_cases.NewGetBillingPlanUseCase(provider)

	_, err := uc.Execute(context.Background(), dto.GetBillingPlanInput{})

	assert.ErrorIs(t, err, apperrors.ErrInvalidSubscriptionRequest)
	assert.False(t, provider.getPlanCalled)
}

func TestGetBillingPlanUseCase_RejectsMissingPlan(t *testing.T) {
	provider := &billingQueriesProviderFake{}
	uc := use_cases.NewGetBillingPlanUseCase(provider)

	_, err := uc.Execute(context.Background(), dto.GetBillingPlanInput{
		PlanID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrBillingPlanNotAvailable)
}

func TestGetBillingPlanUseCase_PropagatesProviderError(t *testing.T) {
	providerErr := errors.New("Billing unavailable")
	provider := &billingQueriesProviderFake{getPlanErr: providerErr}
	uc := use_cases.NewGetBillingPlanUseCase(provider)

	_, err := uc.Execute(context.Background(), dto.GetBillingPlanInput{
		PlanID: uuid.New(),
	})

	assert.ErrorIs(t, err, providerErr)
}

func TestListStoreSubscriptionsUseCase_Success(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	subscriptionID := uuid.New()
	planID := uuid.New()

	provider := &billingQueriesProviderFake{
		subscriptions: []ports.BillingSubscription{
			{
				ID:       subscriptionID,
				StoreID:  store.ID().Value(),
				PlanID:   planID,
				Price:    ports.BillingMoney{AmountMinor: 500000, Currency: "NGN"},
				Duration: 30 * 24 * time.Hour,
				Status:   "active",
			},
		},
	}
	repository := &subscribeStoreRepositoryFake{store: store}
	uc := use_cases.NewListStoreSubscriptionsUseCase(repository, provider)

	output, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.ListStoreSubscriptionsInput{
		StoreID: store.ID().Value(),
	})

	require.NoError(t, err)
	require.Len(t, output, 1)
	assert.True(t, repository.findByIDCalled)
	assert.True(t, provider.listSubscriptionsCalled)
	assert.Equal(t, store.ID().Value(), provider.requestedStoreID)
	assert.Equal(t, subscriptionID, output[0].ID)
	assert.Equal(t, store.ID().Value(), output[0].StoreID)
	assert.Equal(t, int64(500000), output[0].Price.AmountMinor)
	assert.Equal(t, "active", output[0].Status)
}

func TestListStoreSubscriptionsUseCase_RejectsUnauthenticatedRequest(t *testing.T) {
	provider := &billingQueriesProviderFake{}
	repository := &subscribeStoreRepositoryFake{}
	uc := use_cases.NewListStoreSubscriptionsUseCase(repository, provider)

	_, err := uc.Execute(context.Background(), dto.ListStoreSubscriptionsInput{
		StoreID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrUnauthenticated)
	assert.False(t, repository.findByIDCalled)
	assert.False(t, provider.listSubscriptionsCalled)
}

func TestListStoreSubscriptionsUseCase_RejectsNonOwner(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	provider := &billingQueriesProviderFake{}
	repository := &subscribeStoreRepositoryFake{store: store}
	uc := use_cases.NewListStoreSubscriptionsUseCase(repository, provider)

	_, err := uc.Execute(subscribeAuthenticatedContext(uuid.New()), dto.ListStoreSubscriptionsInput{
		StoreID: store.ID().Value(),
	})

	assert.ErrorIs(t, err, apperrors.ErrStoreNotFound)
	assert.False(t, provider.listSubscriptionsCalled)
}

func TestListStoreSubscriptionsUseCase_RejectsNilStoreID(t *testing.T) {
	provider := &billingQueriesProviderFake{}
	repository := &subscribeStoreRepositoryFake{}
	uc := use_cases.NewListStoreSubscriptionsUseCase(repository, provider)

	_, err := uc.Execute(subscribeAuthenticatedContext(uuid.New()), dto.ListStoreSubscriptionsInput{})

	assert.ErrorIs(t, err, apperrors.ErrInvalidSubscriptionRequest)
	assert.False(t, repository.findByIDCalled)
}

func TestListStoreSubscriptionsUseCase_RejectsSubscriptionFromAnotherStore(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	provider := &billingQueriesProviderFake{
		subscriptions: []ports.BillingSubscription{
			{ID: uuid.New(), StoreID: uuid.New()},
		},
	}
	uc := use_cases.NewListStoreSubscriptionsUseCase(
		&subscribeStoreRepositoryFake{store: store},
		provider,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.ListStoreSubscriptionsInput{
		StoreID: store.ID().Value(),
	})

	assert.ErrorIs(t, err, apperrors.ErrBillingSubscriptionNotFound)
}

func TestListStoreSubscriptionsUseCase_PropagatesProviderError(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	providerErr := errors.New("Billing unavailable")
	provider := &billingQueriesProviderFake{listSubscriptionsErr: providerErr}
	uc := use_cases.NewListStoreSubscriptionsUseCase(
		&subscribeStoreRepositoryFake{store: store},
		provider,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.ListStoreSubscriptionsInput{
		StoreID: store.ID().Value(),
	})

	assert.ErrorIs(t, err, providerErr)
}

func TestGetStoreSubscriptionUseCase_Success(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	subscriptionID := uuid.New()
	planID := uuid.New()

	provider := &billingQueriesProviderFake{
		subscription: &ports.BillingSubscription{
			ID:       subscriptionID,
			StoreID:  store.ID().Value(),
			PlanID:   planID,
			Status:   "active",
			Price:    ports.BillingMoney{AmountMinor: 500000, Currency: "NGN"},
			Duration: 30 * 24 * time.Hour,
		},
	}
	repository := &subscribeStoreRepositoryFake{store: store}
	uc := use_cases.NewGetStoreSubscriptionUseCase(repository, provider)

	output, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.GetStoreSubscriptionInput{
		StoreID:        store.ID().Value(),
		SubscriptionID: subscriptionID,
	})

	require.NoError(t, err)
	assert.True(t, repository.findByIDCalled)
	assert.True(t, provider.getSubscriptionCalled)
	assert.Equal(t, subscriptionID, provider.requestedSubscriptionID)
	assert.Equal(t, subscriptionID, output.ID)
	assert.Equal(t, store.ID().Value(), output.StoreID)
	assert.Equal(t, planID, output.PlanID)
}

func TestGetStoreSubscriptionUseCase_RejectsUnauthenticatedRequest(t *testing.T) {
	provider := &billingQueriesProviderFake{}
	repository := &subscribeStoreRepositoryFake{}
	uc := use_cases.NewGetStoreSubscriptionUseCase(repository, provider)

	_, err := uc.Execute(context.Background(), dto.GetStoreSubscriptionInput{
		StoreID:        uuid.New(),
		SubscriptionID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrUnauthenticated)
	assert.False(t, repository.findByIDCalled)
	assert.False(t, provider.getSubscriptionCalled)
}

func TestGetStoreSubscriptionUseCase_RejectsNonOwner(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	provider := &billingQueriesProviderFake{}
	uc := use_cases.NewGetStoreSubscriptionUseCase(
		&subscribeStoreRepositoryFake{store: store},
		provider,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(uuid.New()), dto.GetStoreSubscriptionInput{
		StoreID:        store.ID().Value(),
		SubscriptionID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrStoreNotFound)
	assert.False(t, provider.getSubscriptionCalled)
}

func TestGetStoreSubscriptionUseCase_RejectsSubscriptionBelongingToAnotherStore(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	subscriptionID := uuid.New()
	provider := &billingQueriesProviderFake{
		subscription: &ports.BillingSubscription{
			ID:      subscriptionID,
			StoreID: uuid.New(),
		},
	}
	uc := use_cases.NewGetStoreSubscriptionUseCase(
		&subscribeStoreRepositoryFake{store: store},
		provider,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.GetStoreSubscriptionInput{
		StoreID:        store.ID().Value(),
		SubscriptionID: subscriptionID,
	})

	assert.ErrorIs(t, err, apperrors.ErrBillingSubscriptionNotFound)
}

func TestGetStoreSubscriptionUseCase_RejectsMissingSubscription(t *testing.T) {
	ownerID := uuid.New()
	store := subscribeTestStore(t, ownerID, valueobjects.PlanTypeBasic, 0)
	provider := &billingQueriesProviderFake{}
	uc := use_cases.NewGetStoreSubscriptionUseCase(
		&subscribeStoreRepositoryFake{store: store},
		provider,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(ownerID), dto.GetStoreSubscriptionInput{
		StoreID:        store.ID().Value(),
		SubscriptionID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrBillingSubscriptionNotFound)
}

func TestGetStoreSubscriptionUseCase_RejectsNilIdentifiers(t *testing.T) {
	provider := &billingQueriesProviderFake{}
	uc := use_cases.NewGetStoreSubscriptionUseCase(
		&subscribeStoreRepositoryFake{},
		provider,
	)

	_, err := uc.Execute(subscribeAuthenticatedContext(uuid.New()), dto.GetStoreSubscriptionInput{
		StoreID: uuid.New(),
	})

	assert.ErrorIs(t, err, apperrors.ErrInvalidSubscriptionRequest)
	assert.False(t, provider.getSubscriptionCalled)
}
