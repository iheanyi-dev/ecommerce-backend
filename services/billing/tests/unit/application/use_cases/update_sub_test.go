package use_cases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func newUpdateSubFixture(
	status entities.SubscriptionStatus,
) (*use_cases.UpdateSub, *fakeSubscriptionRepository, *fakeUnitOfWork, *fakeSubscriptionOutboxRepository, uuid.UUID) {
	subscriptionID := uuid.New()
	now := time.Now().UTC().Add(-24 * time.Hour)

	subscription := &entities.Subscription{
		ID:        subscriptionID,
		StoreID:   uuid.New(),
		PlanID:    uuid.New(),
		Price:     value_objects.Money{AmountMinor: 500000, Currency: "NGN"},
		Duration:  30 * 24 * time.Hour,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}

	repository := &fakeSubscriptionRepository{current: subscription}
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{}
	plans := &fakePlanRepository{plan: entities.Plan{ID: subscription.PlanID, Type: "premium"}}

	return use_cases.NewUpdateSub(repository, uow, outbox, plans),
		repository, uow, outbox, subscriptionID
}

func TestUpdateSubActivatesPendingSubscriptionAndRecordsStoreEvent(t *testing.T) {
	useCase, repository, uow, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)

	output, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.Subscription.Status != entities.SubscriptionStatusActive {
		t.Fatalf("status = %q, want active", output.Subscription.Status)
	}
	if output.Subscription.StartedAt.IsZero() || output.Subscription.ExpiresAt.IsZero() {
		t.Fatal("activation timestamps should be set")
	}
	if !output.Subscription.ExpiresAt.Equal(
		output.Subscription.StartedAt.Add(30 * 24 * time.Hour),
	) {
		t.Fatal("expiry should equal activation time plus saved duration")
	}
	if repository.updated == nil {
		t.Fatal("subscription was not persisted")
	}
	if uow.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", uow.calls)
	}
	if outbox.calls != 1 {
		t.Fatalf("outbox calls = %d, want 1", outbox.calls)
	}
	if outbox.eventID == "" || outbox.subscriptionID != subscriptionID.String() {
		t.Fatal("outbox event has invalid identifiers")
	}
	if outbox.eventType != "billing.subscription.updated" {
		t.Fatalf("event type = %q", outbox.eventType)
	}
	if len(outbox.payload) == 0 {
		t.Fatal("outbox payload is empty")
	}

	var event struct {
		PlanType string `json:"plan_type"`
	}
	if err := json.Unmarshal(outbox.payload, &event); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}
	if event.PlanType != "premium" {
		t.Fatalf("event plan_type = %q, want premium", event.PlanType)
	}
}

func TestUpdateSubMarksPendingSubscriptionFailed(t *testing.T) {
	useCase, repository, _, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)

	output, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusFailed,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.Subscription.Status != entities.SubscriptionStatusFailed {
		t.Fatalf("status = %q, want failed", output.Subscription.Status)
	}
	if repository.updated == nil || outbox.calls != 1 {
		t.Fatal("failed subscription must be persisted and recorded in the outbox")
	}
}

func TestUpdateSubDoesNotExtendActiveSubscriptionForRenewalEvent(t *testing.T) {
	useCase, repository, _, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusActive)

	previousExpiry := time.Now().UTC().Add(10 * 24 * time.Hour)
	repository.current.ExpiresAt = previousExpiry

	output, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !output.Subscription.ExpiresAt.Equal(previousExpiry) {
		t.Fatal("duplicate success event must not extend the existing subscription")
	}
	if repository.updated != nil || outbox.calls != 0 {
		t.Fatal("unchanged subscription must not be persisted or recorded again")
	}
}

func TestUpdateSubFailedPaymentDoesNotDeactivateActiveSubscription(t *testing.T) {
	useCase, repository, _, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusActive)

	previousExpiry := time.Now().UTC().Add(10 * 24 * time.Hour)
	repository.current.ExpiresAt = previousExpiry

	output, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusFailed,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.Subscription.Status != entities.SubscriptionStatusActive {
		t.Fatalf("status = %q, want active", output.Subscription.Status)
	}
	if !output.Subscription.ExpiresAt.Equal(previousExpiry) {
		t.Fatal("failed payment must not change the active subscription expiry")
	}
	if repository.updated != nil || outbox.calls != 0 {
		t.Fatal("unchanged subscription must not be persisted or recorded")
	}
}

func TestUpdateSubIsIdempotentForRepeatedFailure(t *testing.T) {
	useCase, repository, _, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusFailed)

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusFailed,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.updated != nil || outbox.calls != 0 {
		t.Fatal("repeated failure must not persist or record another event")
	}
}

func TestUpdateSubRejectsInvalidBusinessReference(t *testing.T) {
	useCase, _, _, _, _ := newUpdateSubFixture(entities.SubscriptionStatusPending)

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: "not-a-uuid",
		Status:              entities.SubscriptionStatusActive,
	})
	if !errors.Is(err, domain.ErrInvalidSubscription) {
		t.Fatalf("error = %v, want ErrInvalidSubscription", err)
	}
}

func TestUpdateSubRejectsUnsupportedStatus(t *testing.T) {
	useCase, _, _, _, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusCancelled,
	})
	if !errors.Is(err, domain.ErrInvalidSubscriptionStatus) {
		t.Fatalf("error = %v, want ErrInvalidSubscriptionStatus", err)
	}
}

func TestUpdateSubReturnsNotFound(t *testing.T) {
	useCase, repository, _, _, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)
	repository.current = nil

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	})
	if !errors.Is(err, application.ErrSubscriptionNotFound) {
		t.Fatalf("error = %v, want ErrSubscriptionNotFound", err)
	}
}

func TestUpdateSubPropagatesRepositoryReadError(t *testing.T) {
	useCase, repository, _, _, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)
	repository.getErr = errors.New("database unavailable")

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	})
	if err == nil {
		t.Fatal("expected repository read error")
	}
}

func TestUpdateSubPropagatesRepositoryUpdateError(t *testing.T) {
	useCase, repository, _, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)
	repository.updateErr = errors.New("database update failed")

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	})
	if err == nil {
		t.Fatal("expected repository update error")
	}
	if outbox.calls != 0 {
		t.Fatal("outbox must not be written when subscription persistence fails")
	}
}

func TestUpdateSubPropagatesOutboxError(t *testing.T) {
	useCase, repository, _, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)
	outbox.err = errors.New("outbox unavailable")

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	})
	if err == nil {
		t.Fatal("expected outbox error")
	}
	if repository.updated == nil {
		t.Fatal("subscription update should have been attempted")
	}
}

func TestUpdateSubPropagatesUnitOfWorkError(t *testing.T) {
	useCase, repository, uow, outbox, subscriptionID :=
		newUpdateSubFixture(entities.SubscriptionStatusPending)
	uow.err = errors.New("transaction unavailable")

	_, err := useCase.Execute(context.Background(), use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	})
	if err == nil {
		t.Fatal("expected unit of work error")
	}
	if repository.updated != nil || outbox.calls != 0 {
		t.Fatal("callback must not execute when transaction cannot start")
	}
}

func TestUpdateSubRejectsMissingDependencies(t *testing.T) {
	subscriptionID := uuid.New()
	input := use_cases.UpdateSubInput{
		BusinessReferenceID: subscriptionID.String(),
		Status:              entities.SubscriptionStatusActive,
	}

	_, err := use_cases.NewUpdateSub(nil, &fakeUnitOfWork{}, &fakeSubscriptionOutboxRepository{}, &fakePlanRepository{}).
		Execute(context.Background(), input)
	if !errors.Is(err, application.ErrSubscriptionRepositoryUnavailable) {
		t.Fatalf("error = %v, want repository unavailable", err)
	}

	_, err = use_cases.NewUpdateSub(
		&fakeSubscriptionRepository{},
		nil,
		&fakeSubscriptionOutboxRepository{},
		&fakePlanRepository{},
	).Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected missing unit of work error")
	}

	_, err = use_cases.NewUpdateSub(
		&fakeSubscriptionRepository{},
		&fakeUnitOfWork{},
		nil,
		&fakePlanRepository{},
	).Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected missing outbox error")
	}
}
