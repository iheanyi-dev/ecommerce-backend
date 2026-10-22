package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

func TestGetMyCartReturnsExistingCart(t *testing.T) {
	customerID := uuid.New()
	cart, _ := entities.NewCart(customerID)
	repo := &cartRepositoryMock{cart: cart}
	logger := &loggerMock{}
	metrics := &metricsMock{}
	uc := use_cases.NewGetMyCart(repo, logger, metrics)

	got, err := uc.Execute(context.Background(), customerID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != cart {
		t.Fatal("Execute() did not return the existing cart")
	}
	if repo.getCalls != 1 || repo.saveCalls != 0 {
		t.Fatalf("repository calls: get=%d save=%d, want 1 and 0", repo.getCalls, repo.saveCalls)
	}
	if len(logger.events) != 1 || logger.events[0].Event != "cart.retrieved" {
		t.Fatalf("unexpected logs: %#v", logger.events)
	}
	if len(metrics.metrics) != 1 || metrics.metrics[0].Labels["result"] != "success" {
		t.Fatalf("unexpected metrics: %#v", metrics.metrics)
	}
}

func TestGetMyCartDoesNotCreateMissingCart(t *testing.T) {
	repo := &cartRepositoryMock{}
	uc := use_cases.NewGetMyCart(repo, &loggerMock{}, &metricsMock{})

	_, err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, domainerrors.ErrCartNotFound) {
		t.Fatalf("Execute() error = %v, want ErrCartNotFound", err)
	}
	if repo.saveCalls != 0 {
		t.Fatal("GetMyCart must not create a cart")
	}
}

func TestGetMyCartRejectsNilCustomerID(t *testing.T) {
	repo := &cartRepositoryMock{}
	uc := use_cases.NewGetMyCart(repo, &loggerMock{}, &metricsMock{})

	_, err := uc.Execute(context.Background(), uuid.Nil)
	if !errors.Is(err, domainerrors.ErrInvalidCustomerID) {
		t.Fatalf("Execute() error = %v, want ErrInvalidCustomerID", err)
	}
	if repo.getCalls != 0 {
		t.Fatal("repository should not be called for an invalid customer ID")
	}
}

func TestGetMyCartPropagatesRepositoryFailure(t *testing.T) {
	repositoryErr := errors.New("storage unavailable")
	repo := &cartRepositoryMock{getErr: repositoryErr}
	logger := &loggerMock{}
	metrics := &metricsMock{}
	uc := use_cases.NewGetMyCart(repo, logger, metrics)

	_, err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Execute() error = %v, want wrapped repository error", err)
	}
	if len(logger.events) != 1 || logger.events[0].FailureCategory != "persistence" {
		t.Fatalf("unexpected logs: %#v", logger.events)
	}
	if len(metrics.metrics) != 1 || metrics.metrics[0].Labels["result"] != "failure" {
		t.Fatalf("unexpected metrics: %#v", metrics.metrics)
	}
}
