package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func newDueSubscription(now time.Time) *entities.Subscription {
	return &entities.Subscription{
		ID:        uuid.New(),
		StoreID:   uuid.New(),
		PlanID:    uuid.New(),
		Price:     value_objects.Money{AmountMinor: 500000, Currency: "NGN"},
		Duration:  30 * 24 * time.Hour,
		Status:    entities.SubscriptionStatusActive,
		StartedAt: now.Add(-31 * 24 * time.Hour),
		ExpiresAt: now,
		CreatedAt: now.Add(-31 * 24 * time.Hour),
		UpdatedAt: now.Add(-31 * 24 * time.Hour),
	}
}

func TestExpireSubscriptionsExpiresDueSubscriptionsAndRecordsEvents(t *testing.T) {
	now := time.Now().UTC()
	first := newDueSubscription(now)
	second := newDueSubscription(now.Add(-time.Hour))
	repository := &fakeSubscriptionRepository{
		subscriptions: []*entities.Subscription{first, second},
	}
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{}
	plans := &fakePlanRepository{plan: entities.Plan{ID: first.PlanID, Type: "premium"}}
	useCase := use_cases.NewExpireSubscriptions(repository, uow, outbox, plans)

	output, err := useCase.Execute(context.Background(), use_cases.ExpireSubscriptionsInput{
		Now:       now,
		BatchSize: 10,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.Expired != 2 {
		t.Fatalf("expired = %d, want 2", output.Expired)
	}
	if uow.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", uow.calls)
	}
	if outbox.calls != 2 {
		t.Fatalf("outbox calls = %d, want 2", outbox.calls)
	}
	if repository.updated == nil {
		t.Fatal("expected repository to persist expired subscriptions")
	}
	if repository.updated.Status != entities.SubscriptionStatusExpired {
		t.Fatalf("persisted status = %q, want expired", repository.updated.Status)
	}
}

func TestExpireSubscriptionsDoesNothingWhenNoSubscriptionsAreDue(t *testing.T) {
	now := time.Now().UTC()
	notDue := newDueSubscription(now.Add(24 * time.Hour))
	repository := &fakeSubscriptionRepository{subscriptions: []*entities.Subscription{notDue}}
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{}
	plans := &fakePlanRepository{plan: entities.Plan{ID: notDue.PlanID, Type: "premium"}}
	useCase := use_cases.NewExpireSubscriptions(repository, uow, outbox, plans)

	output, err := useCase.Execute(context.Background(), use_cases.ExpireSubscriptionsInput{
		Now:       now,
		BatchSize: 10,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.Expired != 0 || outbox.calls != 0 {
		t.Fatal("no events should be recorded when no subscriptions are due")
	}
}

func TestExpireSubscriptionsRollsBackBatchWhenOutboxCreationFails(t *testing.T) {
	now := time.Now().UTC()
	due := newDueSubscription(now)
	repository := &fakeSubscriptionRepository{subscriptions: []*entities.Subscription{due}}
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{err: errors.New("outbox unavailable")}
	plans := &fakePlanRepository{plan: entities.Plan{ID: due.PlanID, Type: "premium"}}
	useCase := use_cases.NewExpireSubscriptions(repository, uow, outbox, plans)

	_, err := useCase.Execute(context.Background(), use_cases.ExpireSubscriptionsInput{
		Now:       now,
		BatchSize: 10,
	})
	if err == nil {
		t.Fatal("expected outbox error")
	}
	if uow.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", uow.calls)
	}
	if outbox.calls != 1 {
		t.Fatalf("outbox calls = %d, want 1", outbox.calls)
	}
}

func TestExpireSubscriptionsRejectsInvalidInput(t *testing.T) {
	due := newDueSubscription(time.Now().UTC())
	useCase := use_cases.NewExpireSubscriptions(
		&fakeSubscriptionRepository{subscriptions: []*entities.Subscription{due}},
		&fakeUnitOfWork{},
		&fakeSubscriptionOutboxRepository{},
		&fakePlanRepository{plan: entities.Plan{ID: due.PlanID, Type: "premium"}},
	)

	tests := []struct {
		name  string
		input use_cases.ExpireSubscriptionsInput
	}{
		{name: "missing time", input: use_cases.ExpireSubscriptionsInput{BatchSize: 10}},
		{name: "negative batch size", input: use_cases.ExpireSubscriptionsInput{Now: time.Now(), BatchSize: -1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := useCase.Execute(context.Background(), test.input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
