package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	billingv1 "github.com/iheanyi-dev/ecommerce-backend/proto/billing/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

type planRepositoryMock struct {
	plans   map[uuid.UUID]entities.Plan
	created *entities.Plan
	updated *entities.Plan
	deleted uuid.UUID
}

func (m *planRepositoryMock) Create(_ context.Context, plan entities.Plan) error {
	m.created = &plan
	m.plans[plan.ID] = plan
	return nil
}

func (m *planRepositoryMock) Update(_ context.Context, plan entities.Plan) error {
	m.updated = &plan
	m.plans[plan.ID] = plan
	return nil
}

func (m *planRepositoryMock) DeleteByID(_ context.Context, id uuid.UUID) error {
	m.deleted = id
	delete(m.plans, id)
	return nil
}

func (m *planRepositoryMock) GetAll(context.Context) ([]entities.Plan, error) {
	result := make([]entities.Plan, 0, len(m.plans))
	for _, plan := range m.plans {
		result = append(result, plan)
	}
	return result, nil
}

func (m *planRepositoryMock) GetByID(_ context.Context, id uuid.UUID) (entities.Plan, error) {
	return m.plans[id], nil
}

func newValidPlan(t *testing.T) entities.Plan {
	t.Helper()

	price, err := value_objects.NewMoney(2500, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	return entities.Plan{
		ID:          uuid.New(),
		Type:        "premium",
		Name:        "Premium",
		Description: "Premium subscription",
		Price:       price,
		Duration:    30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{
			MaxProducts: 5,
			AIEnabled:   true,
		},
		Available: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestCreatePlanRPC(t *testing.T) {
	repository := &planRepositoryMock{plans: make(map[uuid.UUID]entities.Plan)}
	server := &Server{createPlan: use_cases.NewCreatePlan(repository)}

	response, err := server.CreatePlan(context.Background(), &billingv1.CreatePlanRequest{
		Type: "premium", Name: "Premium", Description: "Premium subscription",
		Price:           &billingv1.Money{AmountMinor: 2500, Currency: "NGN"},
		DurationSeconds: int64((30 * 24 * time.Hour) / time.Second),
		Capabilities:    &billingv1.PlanCapabilities{MaxProducts: 5, AiEnabled: true},
		Available:       true,
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if repository.created == nil {
		t.Fatal("plan was not persisted")
	}
	if response.Plan.Id != repository.created.ID.String() ||
		response.Plan.Name != "Premium" ||
		response.Plan.Price.AmountMinor != 2500 ||
		response.Plan.Capabilities.MaxProducts != 5 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestGetPlanRPC(t *testing.T) {
	plan := newValidPlan(t)
	repository := &planRepositoryMock{
		plans: map[uuid.UUID]entities.Plan{plan.ID: plan},
	}
	server := &Server{getPlan: use_cases.NewGetPlanByID(repository)}

	response, err := server.GetPlan(context.Background(), &billingv1.GetPlanRequest{
		Id: plan.ID.String(),
	})
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if response.Plan.Id != plan.ID.String() ||
		response.Plan.Name != plan.Name ||
		response.Plan.Price.Currency != plan.Price.Currency {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestListPlansRPC(t *testing.T) {
	first := newValidPlan(t)
	second := newValidPlan(t)
	repository := &planRepositoryMock{
		plans: map[uuid.UUID]entities.Plan{
			first.ID:  first,
			second.ID: second,
		},
	}
	server := &Server{getPlans: use_cases.NewGetPlans(repository)}

	response, err := server.ListPlans(context.Background(), &billingv1.Empty{})
	if err != nil {
		t.Fatalf("ListPlans() error = %v", err)
	}
	if len(response.Plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(response.Plans))
	}
}

func TestListPlansRPCReturnsEmptyList(t *testing.T) {
	server := &Server{
		getPlans: use_cases.NewGetPlans(
			&planRepositoryMock{plans: make(map[uuid.UUID]entities.Plan)},
		),
	}

	response, err := server.ListPlans(context.Background(), &billingv1.Empty{})
	if err != nil {
		t.Fatalf("ListPlans() error = %v", err)
	}
	if response.Plans == nil || len(response.Plans) != 0 {
		t.Fatalf("expected non-nil empty plans, got %+v", response.Plans)
	}
}

type subscriptionRepositoryMock struct {
	subscriptions map[uuid.UUID]entities.Subscription
}

func (m *subscriptionRepositoryMock) GetByStoreID(_ context.Context, storeID uuid.UUID) ([]entities.Subscription, error) {
	result := make([]entities.Subscription, 0)
	for _, subscription := range m.subscriptions {
		if subscription.StoreID == storeID {
			result = append(result, subscription)
		}
	}
	return result, nil
}

func (m *subscriptionRepositoryMock) GetByID(_ context.Context, id uuid.UUID) (*entities.Subscription, error) {
	subscription, ok := m.subscriptions[id]
	if !ok {
		return nil, nil
	}
	return &subscription, nil
}

func (m *subscriptionRepositoryMock) GetCurrentByStore(context.Context, uuid.UUID) (*entities.Subscription, error) {
	return nil, nil
}

func (m *subscriptionRepositoryMock) Create(_ context.Context, subscription entities.Subscription) error {
	m.subscriptions[subscription.ID] = subscription
	return nil
}

func (m *subscriptionRepositoryMock) Update(_ context.Context, subscription entities.Subscription) error {
	m.subscriptions[subscription.ID] = subscription
	return nil
}

func (m *subscriptionRepositoryMock) GetDueForUpdate(context.Context, time.Time, int32) ([]entities.Subscription, error) {
	return nil, nil
}

func newValidSubscription(t *testing.T, storeID uuid.UUID) entities.Subscription {
	t.Helper()

	price, err := value_objects.NewMoney(2500, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	return entities.Subscription{
		ID:        uuid.New(),
		StoreID:   storeID,
		PlanID:    uuid.New(),
		Price:     price,
		Duration:  30 * 24 * time.Hour,
		Status:    entities.SubscriptionStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestUpdatePlanRPC(t *testing.T) {
	plan := newValidPlan(t)
	repository := &planRepositoryMock{
		plans: map[uuid.UUID]entities.Plan{plan.ID: plan},
	}
	server := &Server{updatePlan: use_cases.NewUpdatePlan(repository)}

	response, err := server.UpdatePlan(context.Background(), &billingv1.UpdatePlanRequest{
		Id:              plan.ID.String(),
		Type:            "premium",
		Name:            "Premium Updated",
		Description:     "Updated description",
		Price:           &billingv1.Money{AmountMinor: 5000, Currency: "NGN"},
		DurationSeconds: int64((30 * 24 * time.Hour) / time.Second),
		Capabilities:    &billingv1.PlanCapabilities{MaxProducts: 10, AiEnabled: true},
		Available:       true,
	})
	if err != nil {
		t.Fatalf("UpdatePlan() error = %v", err)
	}
	if repository.updated == nil {
		t.Fatal("plan was not updated")
	}
	if response.Plan.Name != "Premium Updated" ||
		response.Plan.Price.AmountMinor != 5000 ||
		response.Plan.Capabilities.MaxProducts != 10 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestDeletePlanRPC(t *testing.T) {
	plan := newValidPlan(t)
	repository := &planRepositoryMock{
		plans: map[uuid.UUID]entities.Plan{plan.ID: plan},
	}
	server := &Server{deletePlan: use_cases.NewDeletePlanByID(repository)}

	_, err := server.DeletePlan(context.Background(), &billingv1.DeletePlanRequest{
		Id: plan.ID.String(),
	})
	if err != nil {
		t.Fatalf("DeletePlan() error = %v", err)
	}
	if repository.deleted != plan.ID {
		t.Fatalf("deleted ID = %s, want %s", repository.deleted, plan.ID)
	}
	if _, exists := repository.plans[plan.ID]; exists {
		t.Fatal("plan was not deleted")
	}
}

func TestGetSubscriptionRPC(t *testing.T) {
	storeID := uuid.New()
	subscription := newValidSubscription(t, storeID)
	repository := &subscriptionRepositoryMock{
		subscriptions: map[uuid.UUID]entities.Subscription{
			subscription.ID: subscription,
		},
	}
	server := &Server{getSub: use_cases.NewGetSubByID(repository)}

	response, err := server.GetSubscription(context.Background(), &billingv1.GetSubscriptionRequest{
		SubscriptionId: subscription.ID.String(),
	})
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}
	if response.Subscription.Id != subscription.ID.String() ||
		response.Subscription.StoreId != storeID.String() ||
		response.Subscription.Status != string(subscription.Status) {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestListStoreSubscriptionsRPC(t *testing.T) {
	storeID := uuid.New()
	first := newValidSubscription(t, storeID)
	second := newValidSubscription(t, storeID)
	other := newValidSubscription(t, uuid.New())

	repository := &subscriptionRepositoryMock{
		subscriptions: map[uuid.UUID]entities.Subscription{
			first.ID:  first,
			second.ID: second,
			other.ID:  other,
		},
	}
	server := &Server{getSubs: use_cases.NewGetSubsByStoreID(repository)}

	response, err := server.ListStoreSubscriptions(context.Background(), &billingv1.ListStoreSubscriptionsRequest{
		StoreId: storeID.String(),
	})
	if err != nil {
		t.Fatalf("ListStoreSubscriptions() error = %v", err)
	}
	if len(response.Subscriptions) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(response.Subscriptions))
	}
	for _, subscription := range response.Subscriptions {
		if subscription.StoreId != storeID.String() {
			t.Fatalf("unexpected store ID: %s", subscription.StoreId)
		}
	}
}

func TestListStoreSubscriptionsRPCReturnsEmptyList(t *testing.T) {
	server := &Server{
		getSubs: use_cases.NewGetSubsByStoreID(&subscriptionRepositoryMock{
			subscriptions: make(map[uuid.UUID]entities.Subscription),
		}),
	}

	response, err := server.ListStoreSubscriptions(context.Background(), &billingv1.ListStoreSubscriptionsRequest{
		StoreId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("ListStoreSubscriptions() error = %v", err)
	}
	if response.Subscriptions == nil || len(response.Subscriptions) != 0 {
		t.Fatalf("expected non-nil empty list, got %+v", response.Subscriptions)
	}
}
