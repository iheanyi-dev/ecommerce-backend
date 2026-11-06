package billing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	billingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/billing"
)

func TestProvider_GetPlan(t *testing.T) {
	planID := uuid.New()

	client := &fakeBillingClient{
		getPlanResponse: &billingv1.PlanResponse{
			Plan: &billingv1.Plan{
				Id:        planID.String(),
				Type:      "premium",
				Available: true,
			},
		},
	}

	provider := billing.NewProvider(client, "store-service", "secret")
	plan, err := provider.GetPlan(context.Background(), planID)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, planID, plan.ID)
	assert.Equal(t, "premium", plan.Type)
	assert.True(t, plan.Available)

	require.NotNil(t, client.getPlanRequest)
	assert.Equal(t, planID.String(), client.getPlanRequest.GetId())
	assertServiceMetadata(t, client.getPlanContext)
}

func TestProvider_GetPlan_PropagatesRPCError(t *testing.T) {
	rpcErr := errors.New("Billing unavailable")
	client := &fakeBillingClient{getPlanErr: rpcErr}
	provider := billing.NewProvider(client, "store-service", "secret")

	_, err := provider.GetPlan(context.Background(), uuid.New())

	require.ErrorIs(t, err, rpcErr)
}

func TestProvider_GetPlan_RejectsEmptyResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *billingv1.PlanResponse
	}{
		{name: "nil response"},
		{name: "nil plan", response: &billingv1.PlanResponse{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := billing.NewProvider(
				&fakeBillingClient{getPlanResponse: tt.response},
				"store-service",
				"secret",
			)

			_, err := provider.GetPlan(context.Background(), uuid.New())

			require.Error(t, err)
		})
	}
}

func TestProvider_GetPlan_RejectsInvalidPlanID(t *testing.T) {
	client := &fakeBillingClient{
		getPlanResponse: &billingv1.PlanResponse{
			Plan: &billingv1.Plan{Id: "invalid-id"},
		},
	}
	provider := billing.NewProvider(client, "store-service", "secret")

	_, err := provider.GetPlan(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid plan ID")
}

func TestProvider_Subscribe(t *testing.T) {
	input := ports.BillingSubscribeInput{
		StoreID:       uuid.New(),
		PayerID:       uuid.New(),
		PlanID:        uuid.New(),
		CustomerEmail: "owner@example.com",
	}
	subscriptionID := uuid.New()
	paymentID := uuid.New()

	client := &fakeBillingClient{
		subscribeResponse: &billingv1.SubscribeResponse{
			Subscription: &billingv1.Subscription{Id: subscriptionID.String()},
			PaymentId:    paymentID.String(),
			CheckoutUrl:  "https://checkout.example.com/session",
		},
	}

	provider := billing.NewProvider(client, "store-service", "secret")
	output, err := provider.Subscribe(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, subscriptionID, output.SubscriptionID)
	assert.Equal(t, paymentID, output.PaymentID)
	assert.Equal(t, "https://checkout.example.com/session", output.CheckoutURL)

	require.NotNil(t, client.subscribeRequest)
	assert.Equal(t, input.StoreID.String(), client.subscribeRequest.GetStoreId())
	assert.Equal(t, input.PayerID.String(), client.subscribeRequest.GetPayerId())
	assert.Equal(t, input.PlanID.String(), client.subscribeRequest.GetPlanId())
	assert.Equal(t, input.CustomerEmail, client.subscribeRequest.GetCustomerEmail())
	assertServiceMetadata(t, client.subscribeContext)
}

func TestProvider_Subscribe_PropagatesRPCError(t *testing.T) {
	rpcErr := errors.New("Billing unavailable")
	provider := billing.NewProvider(
		&fakeBillingClient{subscribeErr: rpcErr},
		"store-service",
		"secret",
	)

	_, err := provider.Subscribe(context.Background(), ports.BillingSubscribeInput{
		StoreID: uuid.New(),
		PayerID: uuid.New(),
		PlanID:  uuid.New(),
	})

	require.ErrorIs(t, err, rpcErr)
}

func TestProvider_Subscribe_RejectsEmptyResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *billingv1.SubscribeResponse
	}{
		{name: "nil response"},
		{name: "nil subscription", response: &billingv1.SubscribeResponse{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := billing.NewProvider(
				&fakeBillingClient{subscribeResponse: tt.response},
				"store-service",
				"secret",
			)

			_, err := provider.Subscribe(context.Background(), ports.BillingSubscribeInput{
				StoreID: uuid.New(),
				PayerID: uuid.New(),
				PlanID:  uuid.New(),
			})

			require.Error(t, err)
		})
	}
}

func TestProvider_Subscribe_RejectsInvalidSubscriptionID(t *testing.T) {
	provider := billing.NewProvider(
		&fakeBillingClient{
			subscribeResponse: &billingv1.SubscribeResponse{
				Subscription: &billingv1.Subscription{Id: "invalid-id"},
				PaymentId:    uuid.New().String(),
			},
		},
		"store-service",
		"secret",
	)

	_, err := provider.Subscribe(context.Background(), ports.BillingSubscribeInput{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid subscription ID")
}

func TestProvider_Subscribe_RejectsInvalidPaymentID(t *testing.T) {
	provider := billing.NewProvider(
		&fakeBillingClient{
			subscribeResponse: &billingv1.SubscribeResponse{
				Subscription: &billingv1.Subscription{Id: uuid.New().String()},
				PaymentId:    "invalid-id",
			},
		},
		"store-service",
		"secret",
	)

	_, err := provider.Subscribe(context.Background(), ports.BillingSubscribeInput{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payment ID")
}

func assertServiceMetadata(t *testing.T, ctx context.Context) {
	t.Helper()

	require.NotNil(t, ctx)
	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)

	assert.Equal(t, []string{"store-service"}, md.Get("x-service-name"))
	assert.Equal(t, []string{"secret"}, md.Get("x-service-secret"))
}

type fakeBillingClient struct {
	billingv1.BillingServiceClient

	getPlanRequest  *billingv1.GetPlanRequest
	getPlanContext  context.Context
	getPlanResponse *billingv1.PlanResponse
	getPlanErr      error

	subscribeRequest  *billingv1.SubscribeRequest
	subscribeContext  context.Context
	subscribeResponse *billingv1.SubscribeResponse
	subscribeErr      error

	listPlansContext  context.Context
	listPlansResponse *billingv1.ListPlansResponse
	listPlansErr      error

	getSubscriptionRequest  *billingv1.GetSubscriptionRequest
	getSubscriptionContext  context.Context
	getSubscriptionResponse *billingv1.SubscriptionResponse
	getSubscriptionErr      error

	listSubscriptionsRequest  *billingv1.ListStoreSubscriptionsRequest
	listSubscriptionsContext  context.Context
	listSubscriptionsResponse *billingv1.ListStoreSubscriptionsResponse
	listSubscriptionsErr      error
}

func (f *fakeBillingClient) GetPlan(
	ctx context.Context,
	request *billingv1.GetPlanRequest,
	_ ...grpc.CallOption,
) (*billingv1.PlanResponse, error) {
	f.getPlanContext = ctx
	f.getPlanRequest = request
	return f.getPlanResponse, f.getPlanErr
}

func (f *fakeBillingClient) Subscribe(
	ctx context.Context,
	request *billingv1.SubscribeRequest,
	_ ...grpc.CallOption,
) (*billingv1.SubscribeResponse, error) {
	f.subscribeContext = ctx
	f.subscribeRequest = request
	return f.subscribeResponse, f.subscribeErr
}

func TestProvider_ListPlans(t *testing.T) {
	planID := uuid.New()
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	promotionStart := createdAt.Add(-time.Hour)
	promotionEnd := updatedAt.Add(time.Hour)

	client := &fakeBillingClient{
		listPlansResponse: &billingv1.ListPlansResponse{
			Plans: []*billingv1.Plan{
				{
					Id:          planID.String(),
					Type:        "premium",
					Name:        "Premium",
					Description: "Premium plan",
					Price: &billingv1.Money{
						AmountMinor: 250000,
						Currency:    "NGN",
					},
					DurationSeconds: 2592000,
					Capabilities: &billingv1.PlanCapabilities{
						MaxProducts: 5,
						AiEnabled:   true,
					},
					Available: true,
					Promotion: &billingv1.Promotion{
						Name:                "Launch offer",
						DiscountAmountMinor: 50000,
						StartsAt:            timestamppb.New(promotionStart),
						EndsAt:              timestamppb.New(promotionEnd),
					},
					CreatedAt: timestamppb.New(createdAt),
					UpdatedAt: timestamppb.New(updatedAt),
				},
			},
		},
	}

	provider := billing.NewProvider(client, "store-service", "secret")
	plans, err := provider.ListPlans(context.Background())

	require.NoError(t, err)
	require.Len(t, plans, 1)

	plan := plans[0]
	assert.Equal(t, planID, plan.ID)
	assert.Equal(t, "premium", plan.Type)
	assert.Equal(t, "Premium", plan.Name)
	assert.Equal(t, "Premium plan", plan.Description)
	assert.Equal(t, int64(250000), plan.Price.AmountMinor)
	assert.Equal(t, "NGN", plan.Price.Currency)
	assert.Equal(t, 30*24*time.Hour, plan.Duration)
	assert.Equal(t, 5, plan.Capabilities.MaxProducts)
	assert.True(t, plan.Capabilities.AIEnabled)
	assert.True(t, plan.Available)

	require.NotNil(t, plan.Promotion)
	assert.Equal(t, "Launch offer", plan.Promotion.Name)
	assert.Equal(t, int64(50000), plan.Promotion.DiscountAmountMinor)
	require.NotNil(t, plan.Promotion.StartsAt)
	require.NotNil(t, plan.Promotion.EndsAt)
	assert.Equal(t, promotionStart, *plan.Promotion.StartsAt)
	assert.Equal(t, promotionEnd, *plan.Promotion.EndsAt)

	require.NotNil(t, plan.CreatedAt)
	require.NotNil(t, plan.UpdatedAt)
	assert.Equal(t, createdAt, *plan.CreatedAt)
	assert.Equal(t, updatedAt, *plan.UpdatedAt)
	assertServiceMetadata(t, client.listPlansContext)
}

func TestProvider_ListPlans_ReturnsEmptyList(t *testing.T) {
	provider := billing.NewProvider(
		&fakeBillingClient{
			listPlansResponse: &billingv1.ListPlansResponse{},
		},
		"store-service",
		"secret",
	)

	plans, err := provider.ListPlans(context.Background())

	require.NoError(t, err)
	assert.Empty(t, plans)
}

func TestProvider_ListPlans_RejectsNilResponseAndPlan(t *testing.T) {
	tests := []struct {
		name     string
		response *billingv1.ListPlansResponse
	}{
		{name: "nil response"},
		{name: "nil plan", response: &billingv1.ListPlansResponse{
			Plans: []*billingv1.Plan{nil},
		}},
		{name: "invalid plan ID", response: &billingv1.ListPlansResponse{
			Plans: []*billingv1.Plan{{Id: "invalid"}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := billing.NewProvider(
				&fakeBillingClient{listPlansResponse: tt.response},
				"store-service",
				"secret",
			)

			_, err := provider.ListPlans(context.Background())
			require.Error(t, err)
		})
	}
}

func TestProvider_ListPlans_PropagatesRPCError(t *testing.T) {
	rpcErr := errors.New("Billing unavailable")
	provider := billing.NewProvider(
		&fakeBillingClient{listPlansErr: rpcErr},
		"store-service",
		"secret",
	)

	_, err := provider.ListPlans(context.Background())
	require.ErrorIs(t, err, rpcErr)
}

func TestProvider_GetSubscription(t *testing.T) {
	subscriptionID := uuid.New()
	storeID := uuid.New()
	planID := uuid.New()
	startedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := startedAt.Add(30 * 24 * time.Hour)

	client := &fakeBillingClient{
		getSubscriptionResponse: &billingv1.SubscriptionResponse{
			Subscription: &billingv1.Subscription{
				Id:              subscriptionID.String(),
				StoreId:         storeID.String(),
				PlanId:          planID.String(),
				Price:           &billingv1.Money{AmountMinor: 250000, Currency: "NGN"},
				DurationSeconds: 2592000,
				Status:          "active",
				StartedAt:       timestamppb.New(startedAt),
				ExpiresAt:       timestamppb.New(expiresAt),
				CreatedAt:       timestamppb.New(startedAt),
				UpdatedAt:       timestamppb.New(startedAt),
			},
		},
	}

	provider := billing.NewProvider(client, "store-service", "secret")
	subscription, err := provider.GetSubscription(context.Background(), subscriptionID)

	require.NoError(t, err)
	require.NotNil(t, subscription)
	assert.Equal(t, subscriptionID, subscription.ID)
	assert.Equal(t, storeID, subscription.StoreID)
	assert.Equal(t, planID, subscription.PlanID)
	assert.Equal(t, int64(250000), subscription.Price.AmountMinor)
	assert.Equal(t, "NGN", subscription.Price.Currency)
	assert.Equal(t, 30*24*time.Hour, subscription.Duration)
	assert.Equal(t, "active", subscription.Status)

	require.NotNil(t, subscription.StartedAt)
	require.NotNil(t, subscription.ExpiresAt)
	assert.Equal(t, startedAt, *subscription.StartedAt)
	assert.Equal(t, expiresAt, *subscription.ExpiresAt)

	require.NotNil(t, client.getSubscriptionRequest)
	assert.Equal(t, subscriptionID.String(), client.getSubscriptionRequest.GetSubscriptionId())
	assertServiceMetadata(t, client.getSubscriptionContext)
}

func TestProvider_GetSubscription_RejectsEmptyResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *billingv1.SubscriptionResponse
	}{
		{name: "nil response"},
		{name: "nil subscription", response: &billingv1.SubscriptionResponse{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := billing.NewProvider(
				&fakeBillingClient{getSubscriptionResponse: tt.response},
				"store-service",
				"secret",
			)

			_, err := provider.GetSubscription(context.Background(), uuid.New())
			require.Error(t, err)
		})
	}
}

func TestProvider_GetSubscription_RejectsInvalidIDs(t *testing.T) {
	tests := []struct {
		name         string
		subscription *billingv1.Subscription
	}{
		{name: "subscription ID", subscription: &billingv1.Subscription{
			Id: "invalid", StoreId: uuid.New().String(), PlanId: uuid.New().String(),
		}},
		{name: "Store ID", subscription: &billingv1.Subscription{
			Id: uuid.New().String(), StoreId: "invalid", PlanId: uuid.New().String(),
		}},
		{name: "plan ID", subscription: &billingv1.Subscription{
			Id: uuid.New().String(), StoreId: uuid.New().String(), PlanId: "invalid",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := billing.NewProvider(
				&fakeBillingClient{
					getSubscriptionResponse: &billingv1.SubscriptionResponse{
						Subscription: tt.subscription,
					},
				},
				"store-service",
				"secret",
			)

			_, err := provider.GetSubscription(context.Background(), uuid.New())
			require.Error(t, err)
		})
	}
}

func TestProvider_GetSubscription_PropagatesRPCError(t *testing.T) {
	rpcErr := errors.New("Billing unavailable")
	provider := billing.NewProvider(
		&fakeBillingClient{getSubscriptionErr: rpcErr},
		"store-service",
		"secret",
	)

	_, err := provider.GetSubscription(context.Background(), uuid.New())
	require.ErrorIs(t, err, rpcErr)
}

func TestProvider_ListStoreSubscriptions(t *testing.T) {
	storeID := uuid.New()
	subscriptionID := uuid.New()
	planID := uuid.New()

	client := &fakeBillingClient{
		listSubscriptionsResponse: &billingv1.ListStoreSubscriptionsResponse{
			Subscriptions: []*billingv1.Subscription{
				{
					Id:              subscriptionID.String(),
					StoreId:         storeID.String(),
					PlanId:          planID.String(),
					Price:           &billingv1.Money{AmountMinor: 250000, Currency: "NGN"},
					DurationSeconds: 2592000,
					Status:          "active",
				},
			},
		},
	}

	provider := billing.NewProvider(client, "store-service", "secret")
	subscriptions, err := provider.ListStoreSubscriptions(context.Background(), storeID)

	require.NoError(t, err)
	require.Len(t, subscriptions, 1)
	assert.Equal(t, subscriptionID, subscriptions[0].ID)
	assert.Equal(t, storeID, subscriptions[0].StoreID)
	assert.Equal(t, planID, subscriptions[0].PlanID)
	assert.Equal(t, int64(250000), subscriptions[0].Price.AmountMinor)
	assert.Equal(t, "NGN", subscriptions[0].Price.Currency)
	assert.Equal(t, "active", subscriptions[0].Status)

	require.NotNil(t, client.listSubscriptionsRequest)
	assert.Equal(t, storeID.String(), client.listSubscriptionsRequest.GetStoreId())
	assertServiceMetadata(t, client.listSubscriptionsContext)
}

func TestProvider_ListStoreSubscriptions_ReturnsEmptyList(t *testing.T) {
	provider := billing.NewProvider(
		&fakeBillingClient{
			listSubscriptionsResponse: &billingv1.ListStoreSubscriptionsResponse{},
		},
		"store-service",
		"secret",
	)

	subscriptions, err := provider.ListStoreSubscriptions(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Empty(t, subscriptions)
}

func TestProvider_ListStoreSubscriptions_RejectsInvalidResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *billingv1.ListStoreSubscriptionsResponse
	}{
		{name: "nil response"},
		{name: "nil subscription", response: &billingv1.ListStoreSubscriptionsResponse{
			Subscriptions: []*billingv1.Subscription{nil},
		}},
		{name: "invalid subscription ID", response: &billingv1.ListStoreSubscriptionsResponse{
			Subscriptions: []*billingv1.Subscription{{
				Id: "invalid", StoreId: uuid.New().String(), PlanId: uuid.New().String(),
			}},
		}},
		{name: "invalid Store ID", response: &billingv1.ListStoreSubscriptionsResponse{
			Subscriptions: []*billingv1.Subscription{{
				Id: uuid.New().String(), StoreId: "invalid", PlanId: uuid.New().String(),
			}},
		}},
		{name: "invalid plan ID", response: &billingv1.ListStoreSubscriptionsResponse{
			Subscriptions: []*billingv1.Subscription{{
				Id: uuid.New().String(), StoreId: uuid.New().String(), PlanId: "invalid",
			}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := billing.NewProvider(
				&fakeBillingClient{listSubscriptionsResponse: tt.response},
				"store-service",
				"secret",
			)

			_, err := provider.ListStoreSubscriptions(context.Background(), uuid.New())
			require.Error(t, err)
		})
	}
}

func TestProvider_ListStoreSubscriptions_PropagatesRPCError(t *testing.T) {
	rpcErr := errors.New("Billing unavailable")
	provider := billing.NewProvider(
		&fakeBillingClient{listSubscriptionsErr: rpcErr},
		"store-service",
		"secret",
	)

	_, err := provider.ListStoreSubscriptions(context.Background(), uuid.New())
	require.ErrorIs(t, err, rpcErr)
}

func (f *fakeBillingClient) ListPlans(
	ctx context.Context,
	_ *billingv1.Empty,
	_ ...grpc.CallOption,
) (*billingv1.ListPlansResponse, error) {
	f.listPlansContext = ctx
	return f.listPlansResponse, f.listPlansErr
}

func (f *fakeBillingClient) GetSubscription(
	ctx context.Context,
	request *billingv1.GetSubscriptionRequest,
	_ ...grpc.CallOption,
) (*billingv1.SubscriptionResponse, error) {
	f.getSubscriptionContext = ctx
	f.getSubscriptionRequest = request
	return f.getSubscriptionResponse, f.getSubscriptionErr
}

func (f *fakeBillingClient) ListStoreSubscriptions(
	ctx context.Context,
	request *billingv1.ListStoreSubscriptionsRequest,
	_ ...grpc.CallOption,
) (*billingv1.ListStoreSubscriptionsResponse, error) {
	f.listSubscriptionsContext = ctx
	f.listSubscriptionsRequest = request
	return f.listSubscriptionsResponse, f.listSubscriptionsErr
}
