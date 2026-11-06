package use_cases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

func newCancelSubFixture(
	status entities.SubscriptionStatus,
) (*use_cases.CancelSub, *fakeSubscriptionRepository, *fakeUnitOfWork, *fakeSubscriptionOutboxRepository, entities.Subscription) {
	existing := entities.Subscription{
		ID:        uuid.New(),
		StoreID:   uuid.New(),
		PlanID:    uuid.New(),
		Status:    status,
		CreatedAt: time.Now().UTC(),
	}
	if status == entities.SubscriptionStatusActive {
		existing.StartedAt = time.Now().UTC()
		existing.ExpiresAt = existing.StartedAt.Add(30 * 24 * time.Hour)
	}

	repository := &fakeSubscriptionRepository{current: &existing}
	uow := &fakeUnitOfWork{}
	outbox := &fakeSubscriptionOutboxRepository{}
	plans := &fakePlanRepository{plan: entities.Plan{ID: existing.PlanID, Type: "premium"}}

	return use_cases.NewCancelSub(repository, uow, outbox, plans),
		repository, uow, outbox, existing
}

func TestCancelSubCancelsPendingSubscriptionAndRecordsStoreEvent(t *testing.T) {
	useCase, repository, uow, outbox, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)

	result, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: existing.ID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != entities.SubscriptionStatusCancelled {
		t.Fatalf("status = %q, want cancelled", result.Status)
	}
	if repository.updated == nil ||
		repository.updated.Status != entities.SubscriptionStatusCancelled {
		t.Fatal("cancelled subscription was not persisted")
	}
	if uow.calls != 1 || outbox.calls != 1 {
		t.Fatalf("transaction calls = %d, outbox calls = %d; want 1 each", uow.calls, outbox.calls)
	}
	if outbox.subscriptionID != existing.ID.String() ||
		outbox.eventType != "billing.subscription.updated" {
		t.Fatal("outbox event does not identify the cancelled subscription")
	}
}

func TestCancelSubCancelsActiveSubscription(t *testing.T) {
	useCase, repository, _, outbox, existing :=
		newCancelSubFixture(entities.SubscriptionStatusActive)

	result, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: existing.ID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != entities.SubscriptionStatusCancelled {
		t.Fatalf("status = %q, want cancelled", result.Status)
	}
	if repository.updated == nil || outbox.calls != 1 {
		t.Fatal("cancellation must be persisted and recorded")
	}
}

func TestCancelSubRejectsMissingSubscription(t *testing.T) {
	useCase, repository, _, _, _ :=
		newCancelSubFixture(entities.SubscriptionStatusPending)
	repository.current = nil

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        uuid.New(),
		SubscriptionID: uuid.New(),
	})
	if !errors.Is(err, application.ErrSubscriptionNotFound) {
		t.Fatalf("error = %v, want ErrSubscriptionNotFound", err)
	}
	if repository.updated != nil {
		t.Fatal("must not update a missing subscription")
	}
}

func TestCancelSubRejectsSubscriptionOwnedByAnotherStore(t *testing.T) {
	useCase, repository, _, _, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        uuid.New(),
		SubscriptionID: existing.ID,
	})
	if !errors.Is(err, application.ErrSubscriptionNotFound) {
		t.Fatalf("error = %v, want ErrSubscriptionNotFound", err)
	}
	if repository.updated != nil {
		t.Fatal("must not update another store's subscription")
	}
}

func TestCancelSubRejectsMismatchedSubscriptionID(t *testing.T) {
	useCase, repository, _, _, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: uuid.New(),
	})
	if !errors.Is(err, application.ErrSubscriptionNotFound) {
		t.Fatalf("error = %v, want ErrSubscriptionNotFound", err)
	}
	if repository.updated != nil {
		t.Fatal("must not update a different subscription")
	}
}

func TestCancelSubRejectsInvalidSubscriptionStatus(t *testing.T) {
	useCase, repository, _, _, existing :=
		newCancelSubFixture(entities.SubscriptionStatusFailed)

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: existing.ID,
	})
	if !errors.Is(err, domain.ErrInvalidSubscriptionStatus) {
		t.Fatalf("error = %v, want ErrInvalidSubscriptionStatus", err)
	}
	if repository.updated != nil {
		t.Fatal("must not persist an invalid cancellation")
	}
}

func TestCancelSubPropagatesRepositoryReadError(t *testing.T) {
	useCase, repository, _, _, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)
	repository.getErr = errors.New("database unavailable")

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: existing.ID,
	})
	if err == nil {
		t.Fatal("expected repository read error")
	}
}

func TestCancelSubPropagatesRepositoryUpdateError(t *testing.T) {
	useCase, repository, _, outbox, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)
	repository.updateErr = errors.New("database update failed")

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: existing.ID,
	})
	if err == nil {
		t.Fatal("expected repository update error")
	}
	if outbox.calls != 0 {
		t.Fatal("outbox must not be written when persistence fails")
	}
}

func TestCancelSubPropagatesOutboxError(t *testing.T) {
	useCase, repository, _, outbox, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)
	outbox.err = errors.New("outbox unavailable")

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: existing.ID,
	})
	if err == nil {
		t.Fatal("expected outbox error")
	}
	if repository.updated == nil {
		t.Fatal("subscription update should have been attempted")
	}
}

func TestCancelSubPropagatesUnitOfWorkError(t *testing.T) {
	useCase, repository, uow, outbox, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)
	uow.err = errors.New("transaction unavailable")

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        existing.StoreID,
		SubscriptionID: existing.ID,
	})
	if err == nil {
		t.Fatal("expected unit of work error")
	}
	if repository.updated != nil || outbox.calls != 0 {
		t.Fatal("callback must not execute when transaction cannot start")
	}
}

func TestCancelSubRejectsInvalidInput(t *testing.T) {
	useCase, repository, _, _, existing :=
		newCancelSubFixture(entities.SubscriptionStatusPending)

	_, err := useCase.Execute(context.Background(), use_cases.CancelSubInput{
		StoreID:        uuid.Nil,
		SubscriptionID: existing.ID,
	})
	if !errors.Is(err, domain.ErrInvalidSubscription) {
		t.Fatalf("error = %v, want ErrInvalidSubscription", err)
	}
	if repository.getCalls != 0 {
		t.Fatal("must not query repository with invalid input")
	}
}

func TestCancelSubRejectsMissingDependencies(t *testing.T) {
	input := use_cases.CancelSubInput{
		StoreID:        uuid.New(),
		SubscriptionID: uuid.New(),
	}

	_, err := use_cases.NewCancelSub(nil, &fakeUnitOfWork{}, &fakeSubscriptionOutboxRepository{}, &fakePlanRepository{}).
		Execute(context.Background(), input)
	if !errors.Is(err, application.ErrSubscriptionRepositoryUnavailable) {
		t.Fatalf("error = %v, want repository unavailable", err)
	}

	_, err = use_cases.NewCancelSub(
		&fakeSubscriptionRepository{},
		nil,
		&fakeSubscriptionOutboxRepository{},
		&fakePlanRepository{},
	).Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected missing unit of work error")
	}

	_, err = use_cases.NewCancelSub(
		&fakeSubscriptionRepository{},
		&fakeUnitOfWork{},
		nil,
		&fakePlanRepository{},
	).Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected missing outbox error")
	}
}
