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

func TestClearCartEmptiesAndPersistsExistingCart(t *testing.T) {
	customerID := uuid.New()
	cart, err := entities.NewCart(customerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := cart.AddItem(uuid.New(), uuid.New(), 3); err != nil {
		t.Fatal(err)
	}
	cartID := cart.ID()

	repo := &cartRepositoryMock{cart: cart}
	logger := &loggerMock{}
	metrics := &metricsMock{}
	uc := use_cases.NewClearCart(repo, logger, metrics)

	err = uc.Execute(context.Background(), customerID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(repo.cart.Items()) != 0 {
		t.Fatalf("expected empty cart, got %d items", len(repo.cart.Items()))
	}
	if repo.cart.ID() != cartID {
		t.Fatal("clearing cart changed its ID")
	}
	if repo.cart.CustomerID() != customerID {
		t.Fatal("clearing cart changed its customer ID")
	}
	if repo.saveCalls != 1 {
		t.Fatalf("Save calls = %d, want 1", repo.saveCalls)
	}
	if len(logger.events) != 1 || logger.events[0].Event != "cart.cleared" {
		t.Fatalf("unexpected log events: %+v", logger.events)
	}
	if len(metrics.metrics) != 1 {
		t.Fatalf("metrics recorded = %d, want 1", len(metrics.metrics))
	}
}

func TestClearCartRejectsInvalidCustomerID(t *testing.T) {
	uc := use_cases.NewClearCart(
		&cartRepositoryMock{}, &loggerMock{}, &metricsMock{},
	)

	err := uc.Execute(context.Background(), uuid.Nil)
	if !errors.Is(err, domainerrors.ErrInvalidCustomerID) {
		t.Fatalf("error = %v, want ErrInvalidCustomerID", err)
	}
}

func TestClearCartReturnsCartNotFound(t *testing.T) {
	repo := &cartRepositoryMock{getErr: domainerrors.ErrCartNotFound}
	uc := use_cases.NewClearCart(
		repo, &loggerMock{}, &metricsMock{},
	)

	err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, domainerrors.ErrCartNotFound) {
		t.Fatalf("error = %v, want ErrCartNotFound", err)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("Save calls = %d, want 0", repo.saveCalls)
	}
}

func TestClearCartReturnsSaveFailure(t *testing.T) {
	customerID := uuid.New()
	cart, err := entities.NewCart(customerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := cart.AddItem(uuid.New(), uuid.New(), 1); err != nil {
		t.Fatal(err)
	}

	repo := &cartRepositoryMock{
		cart:    cart,
		saveErr: errors.New("redis unavailable"),
	}
	logger := &loggerMock{}
	metrics := &metricsMock{}
	uc := use_cases.NewClearCart(repo, logger, metrics)

	err = uc.Execute(context.Background(), customerID)
	if err == nil {
		t.Fatal("expected save failure")
	}
	if len(logger.events) == 0 || logger.events[len(logger.events)-1].FailureCategory != "repository" {
		t.Fatalf("expected repository failure log, got %+v", logger.events)
	}
	if len(metrics.metrics) == 0 {
		t.Fatal("expected failure metric")
	}
}
