package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	billingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type paymentMock struct {
	result ports.InitiatePaymentResult
}

func (m *paymentMock) InitiatePayment(
	context.Context,
	ports.InitiatePaymentRequest,
) (ports.InitiatePaymentResult, error) {
	return m.result, nil
}

type unitOfWorkMock struct{}

func (unitOfWorkMock) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

type subscriptionOutboxMock struct {
	created int
}

func (m *subscriptionOutboxMock) Create(
	context.Context,
	string,
	string,
	string,
	[]byte,
) error {
	m.created++
	return nil
}

func (*subscriptionOutboxMock) GetPending(context.Context, int32) ([]ports.SubscriptionOutboxEvent, error) {
	return nil, nil
}

func (*subscriptionOutboxMock) MarkPublished(context.Context, string) error {
	return nil
}

func (*subscriptionOutboxMock) RecordFailure(context.Context, string, error) error {
	return nil
}

type currentSubscriptionRepositoryMock struct {
	*subscriptionRepositoryMock
	current *entities.Subscription
}

func (m *currentSubscriptionRepositoryMock) GetCurrentByStore(
	_ context.Context,
	storeID uuid.UUID,
) (*entities.Subscription, error) {
	if m.current == nil || m.current.StoreID != storeID {
		return nil, nil
	}
	return m.current, nil
}

func TestSubscribeRPC(t *testing.T) {
	plan := newValidPlan(t)
	plans := &planRepositoryMock{
		plans: map[uuid.UUID]entities.Plan{plan.ID: plan},
	}
	subscriptions := &currentSubscriptionRepositoryMock{
		subscriptionRepositoryMock: &subscriptionRepositoryMock{
			subscriptions: make(map[uuid.UUID]entities.Subscription),
		},
	}
	payment := &paymentMock{
		result: ports.InitiatePaymentResult{
			PaymentID:   uuid.NewString(),
			CheckoutURL: "https://checkout.example.test/payment",
		},
	}
	server := &Server{
		subscribe: use_cases.NewSubscribe(
			plans,
			subscriptions,
			payment,
			unitOfWorkMock{},
			&subscriptionOutboxMock{},
		),
	}

	storeID := uuid.New()
	payerID := uuid.New()

	response, err := server.Subscribe(context.Background(), &billingv1.SubscribeRequest{
		StoreId:       storeID.String(),
		PayerId:       payerID.String(),
		PlanId:        plan.ID.String(),
		CustomerEmail: "customer@example.com",
	})
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if response.Subscription == nil {
		t.Fatal("subscription response is nil")
	}
	if response.Subscription.StoreId != storeID.String() {
		t.Fatalf("store ID = %s, want %s", response.Subscription.StoreId, storeID)
	}
	if response.Subscription.PlanId != plan.ID.String() {
		t.Fatalf("plan ID = %s, want %s", response.Subscription.PlanId, plan.ID)
	}
	if response.Subscription.Status != string(entities.SubscriptionStatusPending) {
		t.Fatalf("subscription status = %s, want pending", response.Subscription.Status)
	}
	if response.PaymentId != payment.result.PaymentID {
		t.Fatalf("payment ID = %s, want %s", response.PaymentId, payment.result.PaymentID)
	}
	if response.CheckoutUrl != payment.result.CheckoutURL {
		t.Fatalf("checkout URL = %s, want %s", response.CheckoutUrl, payment.result.CheckoutURL)
	}
	if len(subscriptions.subscriptions) != 1 {
		t.Fatalf("persisted subscriptions = %d, want 1", len(subscriptions.subscriptions))
	}
}

func TestCancelSubscriptionRPC(t *testing.T) {
	storeID := uuid.New()
	subscription := newValidSubscription(t, storeID)
	plan := newValidPlan(t)
	subscription.PlanID = plan.ID
	subscription.Status = entities.SubscriptionStatusActive
	subscription.StartedAt = time.Now().UTC()
	subscription.ExpiresAt = time.Now().UTC().Add(30 * 24 * time.Hour)

	repository := &currentSubscriptionRepositoryMock{
		subscriptionRepositoryMock: &subscriptionRepositoryMock{
			subscriptions: map[uuid.UUID]entities.Subscription{
				subscription.ID: subscription,
			},
		},
		current: &subscription,
	}
	outbox := &subscriptionOutboxMock{}
	server := &Server{
		cancelSub: use_cases.NewCancelSub(
			repository,
			unitOfWorkMock{},
			outbox,
			&planRepositoryMock{plans: map[uuid.UUID]entities.Plan{plan.ID: plan}},
		),
	}

	response, err := server.CancelSubscription(context.Background(), &billingv1.CancelSubscriptionRequest{
		StoreId:        storeID.String(),
		SubscriptionId: subscription.ID.String(),
	})
	if err != nil {
		t.Fatalf("CancelSubscription() error = %v", err)
	}
	if response.Subscription == nil {
		t.Fatal("subscription response is nil")
	}
	if response.Subscription.Id != subscription.ID.String() {
		t.Fatalf("subscription ID = %s, want %s", response.Subscription.Id, subscription.ID)
	}
	if response.Subscription.Status != string(entities.SubscriptionStatusCancelled) {
		t.Fatalf("subscription status = %s, want cancelled", response.Subscription.Status)
	}
	if outbox.created != 1 {
		t.Fatalf("outbox events created = %d, want 1", outbox.created)
	}
	if updated := repository.subscriptions[subscription.ID]; updated.Status != entities.SubscriptionStatusCancelled {
		t.Fatalf("persisted subscription status = %s, want cancelled", updated.Status)
	}
}

func TestSubscribeRPCRejectsInvalidIDs(t *testing.T) {
	server := &Server{}

	_, err := server.Subscribe(context.Background(), &billingv1.SubscribeRequest{
		StoreId:       "invalid",
		PayerId:       uuid.NewString(),
		PlanId:        uuid.NewString(),
		CustomerEmail: "customer@example.com",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Subscribe() code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestCancelSubscriptionRPCRejectsInvalidIDs(t *testing.T) {
	server := &Server{}

	_, err := server.CancelSubscription(context.Background(), &billingv1.CancelSubscriptionRequest{
		StoreId:        uuid.NewString(),
		SubscriptionId: "invalid",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CancelSubscription() code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}
