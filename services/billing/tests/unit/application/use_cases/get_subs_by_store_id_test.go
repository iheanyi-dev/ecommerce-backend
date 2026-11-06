package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

func TestGetSubsByStoreIDReturnsStoreSubscriptionHistory(t *testing.T) {
	storeID := uuid.New()
	otherStoreID := uuid.New()

	first := &entities.Subscription{ID: uuid.New(), StoreID: storeID}
	second := &entities.Subscription{ID: uuid.New(), StoreID: storeID}
	other := &entities.Subscription{ID: uuid.New(), StoreID: otherStoreID}

	repository := &fakeSubscriptionRepository{
		subscriptions: []*entities.Subscription{first, second, other},
	}
	useCase := use_cases.NewGetSubsByStoreID(repository)

	output, err := useCase.Execute(context.Background(), use_cases.GetSubsByStoreIDInput{
		StoreID: storeID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(output.Subscriptions) != 2 {
		t.Fatalf("subscription count = %d, want 2", len(output.Subscriptions))
	}
	for _, subscription := range output.Subscriptions {
		if subscription.StoreID != storeID {
			t.Fatalf("returned subscription belonging to another store: %s", subscription.StoreID)
		}
	}
}

func TestGetSubsByStoreIDReturnsEmptySliceWhenNoHistoryExists(t *testing.T) {
	useCase := use_cases.NewGetSubsByStoreID(&fakeSubscriptionRepository{})

	output, err := useCase.Execute(context.Background(), use_cases.GetSubsByStoreIDInput{
		StoreID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.Subscriptions == nil {
		t.Fatal("expected an empty slice, got nil")
	}
	if len(output.Subscriptions) != 0 {
		t.Fatalf("subscription count = %d, want 0", len(output.Subscriptions))
	}
}

func TestGetSubsByStoreIDRejectsNilStoreID(t *testing.T) {
	useCase := use_cases.NewGetSubsByStoreID(&fakeSubscriptionRepository{})

	_, err := useCase.Execute(context.Background(), use_cases.GetSubsByStoreIDInput{})
	if !errors.Is(err, domain.ErrInvalidSubscription) {
		t.Fatalf("error = %v, want ErrInvalidSubscription", err)
	}
}

func TestGetSubsByStoreIDPropagatesRepositoryError(t *testing.T) {
	repository := &fakeSubscriptionRepository{
		getErr: errors.New("database unavailable"),
	}
	useCase := use_cases.NewGetSubsByStoreID(repository)

	_, err := useCase.Execute(context.Background(), use_cases.GetSubsByStoreIDInput{
		StoreID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected repository error")
	}
}

func TestGetSubsByStoreIDRejectsMissingRepository(t *testing.T) {
	_, err := use_cases.NewGetSubsByStoreID(nil).Execute(
		context.Background(),
		use_cases.GetSubsByStoreIDInput{StoreID: uuid.New()},
	)
	if !errors.Is(err, application.ErrSubscriptionRepositoryUnavailable) {
		t.Fatalf("error = %v, want ErrSubscriptionRepositoryUnavailable", err)
	}
}
