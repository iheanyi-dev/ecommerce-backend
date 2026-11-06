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

func TestGetSubByIDReturnsSubscription(t *testing.T) {
	subscription := &entities.Subscription{
		ID:      uuid.New(),
		StoreID: uuid.New(),
		Status:  entities.SubscriptionStatusActive,
	}
	repository := &fakeSubscriptionRepository{current: subscription}
	useCase := use_cases.NewGetSubByID(repository)

	output, err := useCase.Execute(context.Background(), use_cases.GetSubByIDInput{
		SubscriptionID: subscription.ID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if output.Subscription.ID != subscription.ID {
		t.Fatalf("subscription ID = %s, want %s", output.Subscription.ID, subscription.ID)
	}
	if output.Subscription.StoreID != subscription.StoreID {
		t.Fatal("returned subscription has an unexpected store ID")
	}
}

func TestGetSubByIDReturnsNotFound(t *testing.T) {
	useCase := use_cases.NewGetSubByID(&fakeSubscriptionRepository{})

	_, err := useCase.Execute(context.Background(), use_cases.GetSubByIDInput{
		SubscriptionID: uuid.New(),
	})
	if !errors.Is(err, application.ErrSubscriptionNotFound) {
		t.Fatalf("error = %v, want ErrSubscriptionNotFound", err)
	}
}

func TestGetSubByIDRejectsNilSubscriptionID(t *testing.T) {
	useCase := use_cases.NewGetSubByID(&fakeSubscriptionRepository{})

	_, err := useCase.Execute(context.Background(), use_cases.GetSubByIDInput{})
	if !errors.Is(err, domain.ErrInvalidSubscription) {
		t.Fatalf("error = %v, want ErrInvalidSubscription", err)
	}
}

func TestGetSubByIDPropagatesRepositoryError(t *testing.T) {
	repository := &fakeSubscriptionRepository{
		getErr: errors.New("database unavailable"),
	}
	useCase := use_cases.NewGetSubByID(repository)

	_, err := useCase.Execute(context.Background(), use_cases.GetSubByIDInput{
		SubscriptionID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected repository error")
	}
}

func TestGetSubByIDRejectsMismatchedRepositoryResult(t *testing.T) {
	repository := &fakeSubscriptionRepository{
		current: &entities.Subscription{
			ID:      uuid.New(),
			StoreID: uuid.New(),
		},
	}
	useCase := use_cases.NewGetSubByID(repository)

	_, err := useCase.Execute(context.Background(), use_cases.GetSubByIDInput{
		SubscriptionID: uuid.New(),
	})
	if !errors.Is(err, application.ErrSubscriptionNotFound) {
		t.Fatalf("error = %v, want ErrSubscriptionNotFound", err)
	}
}

func TestGetSubByIDRejectsMissingRepository(t *testing.T) {
	_, err := use_cases.NewGetSubByID(nil).Execute(
		context.Background(),
		use_cases.GetSubByIDInput{SubscriptionID: uuid.New()},
	)
	if !errors.Is(err, application.ErrSubscriptionRepositoryUnavailable) {
		t.Fatalf("error = %v, want ErrSubscriptionRepositoryUnavailable", err)
	}
}
