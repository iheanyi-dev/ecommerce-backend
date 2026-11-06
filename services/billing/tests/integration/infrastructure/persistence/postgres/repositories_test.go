package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/infrastructure/persistence/postgres"
)

func TestPlanRepositoryIntegration(t *testing.T) {
	pool := newBillingTestPool(t)
	ctx := context.Background()

	repository, err := postgres.NewPlanRepository(pool)
	if err != nil {
		t.Fatalf("create plan repository: %v", err)
	}

	plan := newBillingTestPlan(t)
	if err := repository.Create(ctx, plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	got, err := repository.GetByID(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if got.ID != plan.ID || got.Type != plan.Type || got.Price != plan.Price {
		t.Fatalf("retrieved plan does not match: got %+v", got)
	}
	if got.Capabilities != plan.Capabilities {
		t.Fatalf("plan capabilities = %+v, want %+v", got.Capabilities, plan.Capabilities)
	}

	plans, err := repository.GetAll(ctx)
	if err != nil {
		t.Fatalf("get all plans: %v", err)
	}
	if len(plans) != 1 || plans[0].ID != plan.ID {
		t.Fatalf("unexpected plan list: %+v", plans)
	}

	plan.Name = "Updated Premium"
	plan.UpdatedAt = time.Now().UTC()
	if err := repository.Update(ctx, plan); err != nil {
		t.Fatalf("update plan: %v", err)
	}

	got, err = repository.GetByID(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get updated plan: %v", err)
	}
	if got.Name != plan.Name {
		t.Fatalf("updated plan name = %q, want %q", got.Name, plan.Name)
	}

	if err := repository.DeleteByID(ctx, plan.ID); err != nil {
		t.Fatalf("delete plan: %v", err)
	}
	if _, err := repository.GetByID(ctx, plan.ID); !errors.Is(err, application.ErrPlanNotFound) {
		t.Fatalf("expected ErrPlanNotFound after deletion, got %v", err)
	}
}

func TestSubscriptionRepositoryIntegration(t *testing.T) {
	pool := newBillingTestPool(t)
	ctx := context.Background()

	plans, err := postgres.NewPlanRepository(pool)
	if err != nil {
		t.Fatalf("create plan repository: %v", err)
	}
	subscriptions, err := postgres.NewSubscriptionRepository(pool)
	if err != nil {
		t.Fatalf("create subscription repository: %v", err)
	}

	plan := newBillingTestPlan(t)
	if err := plans.Create(ctx, plan); err != nil {
		t.Fatalf("create parent plan: %v", err)
	}

	subscription := newBillingTestSubscription(t, plan)
	if err := subscriptions.Create(ctx, subscription); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	got, err := subscriptions.GetByID(ctx, subscription.ID)
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	if got.ID != subscription.ID || got.StoreID != subscription.StoreID ||
		got.PlanID != subscription.PlanID || got.Price != subscription.Price {
		t.Fatalf("retrieved subscription does not match: got %+v", got)
	}

	current, err := subscriptions.GetCurrentByStore(ctx, subscription.StoreID)
	if err != nil {
		t.Fatalf("get current subscription: %v", err)
	}
	if current.ID != subscription.ID {
		t.Fatalf("current subscription ID = %s, want %s", current.ID, subscription.ID)
	}

	list, err := subscriptions.GetByStoreID(ctx, subscription.StoreID)
	if err != nil {
		t.Fatalf("list store subscriptions: %v", err)
	}
	if len(list) != 1 || list[0].ID != subscription.ID {
		t.Fatalf("unexpected subscription list: %+v", list)
	}

	subscription.Status = "active"
	subscription.StartedAt = time.Now().UTC().Truncate(time.Microsecond)
	subscription.ExpiresAt = subscription.StartedAt.Add(subscription.Duration)
	subscription.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)

	if err := subscriptions.Update(ctx, subscription); err != nil {
		t.Fatalf("update subscription: %v", err)
	}

	got, err = subscriptions.GetByID(ctx, subscription.ID)
	if err != nil {
		t.Fatalf("get updated subscription: %v", err)
	}
	if got.Status != subscription.Status ||
		!got.StartedAt.Equal(subscription.StartedAt) ||
		!got.ExpiresAt.Equal(subscription.ExpiresAt) ||
		!got.UpdatedAt.Equal(subscription.UpdatedAt) {
		t.Fatalf("subscription update was not persisted: got %+v", got)
	}

	if _, err := subscriptions.GetByID(ctx, uuid.New()); !errors.Is(err, application.ErrSubscriptionNotFound) {
		t.Fatalf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestSubscriptionOutboxRepositoryIntegration(t *testing.T) {
	pool := newBillingTestPool(t)
	ctx := context.Background()

	plans, err := postgres.NewPlanRepository(pool)
	if err != nil {
		t.Fatalf("create plan repository: %v", err)
	}
	subscriptions, err := postgres.NewSubscriptionRepository(pool)
	if err != nil {
		t.Fatalf("create subscription repository: %v", err)
	}
	outbox, err := postgres.NewSubscriptionOutboxRepository(pool)
	if err != nil {
		t.Fatalf("create outbox repository: %v", err)
	}

	plan := newBillingTestPlan(t)
	if err := plans.Create(ctx, plan); err != nil {
		t.Fatalf("create parent plan: %v", err)
	}
	subscription := newBillingTestSubscription(t, plan)
	if err := subscriptions.Create(ctx, subscription); err != nil {
		t.Fatalf("create parent subscription: %v", err)
	}

	eventID := uuid.New()
	payload, err := json.Marshal(map[string]string{
		"event_id":        eventID.String(),
		"subscription_id": subscription.ID.String(),
		"store_id":        subscription.StoreID.String(),
		"status":          "pending",
	})
	if err != nil {
		t.Fatalf("marshal event payload: %v", err)
	}

	if err := outbox.Create(
		ctx,
		eventID.String(),
		subscription.ID.String(),
		"billing.subscription.updated",
		payload,
	); err != nil {
		t.Fatalf("create outbox event: %v", err)
	}

	pending, err := outbox.GetPending(ctx, 10)
	if err != nil {
		t.Fatalf("get pending events: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != eventID.String() {
		t.Fatalf("unexpected pending events: %+v", pending)
	}
	var gotPayload struct {
		EventID        string `json:"event_id"`
		SubscriptionID string `json:"subscription_id"`
		StoreID        string `json:"store_id"`
		Status         string `json:"status"`
	}
	if err := json.Unmarshal(pending[0].Payload, &gotPayload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}

	var wantPayload struct {
		EventID        string `json:"event_id"`
		SubscriptionID string `json:"subscription_id"`
		StoreID        string `json:"store_id"`
		Status         string `json:"status"`
	}
	if err := json.Unmarshal(payload, &wantPayload); err != nil {
		t.Fatalf("decode expected payload: %v", err)
	}

	if gotPayload != wantPayload {
		t.Fatalf("outbox payload = %+v, want %+v", gotPayload, wantPayload)
	}

	if err := outbox.RecordFailure(ctx, eventID.String(), errors.New("broker unavailable")); err != nil {
		t.Fatalf("record delivery failure: %v", err)
	}

	pending, err = outbox.GetPending(ctx, 10)
	if err != nil {
		t.Fatalf("get pending events after failure: %v", err)
	}
	if len(pending) != 1 || pending[0].Attempts != 1 {
		t.Fatalf("expected one event with one attempt, got %+v", pending)
	}

	if err := outbox.MarkPublished(ctx, eventID.String()); err != nil {
		t.Fatalf("mark event published: %v", err)
	}

	pending, err = outbox.GetPending(ctx, 10)
	if err != nil {
		t.Fatalf("get pending events after publishing: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending events, got %+v", pending)
	}
}
