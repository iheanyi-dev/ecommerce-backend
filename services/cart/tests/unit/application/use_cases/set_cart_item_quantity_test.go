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

func TestSetCartItemQuantityUpdatesAndPersists(t *testing.T) {
	customerID, productID, storeID := uuid.New(), uuid.New(), uuid.New()
	cart, _ := entities.NewCart(customerID)
	if err := cart.AddItem(productID, storeID, 2); err != nil {
		t.Fatal(err)
	}
	itemID := cart.Items()[0].ID()

	repo := &cartRepositoryMock{cart: cart}
	logger := &loggerMock{}
	metrics := &metricsMock{}
	uc := use_cases.NewSetCartItemQuantity(repo, logger, metrics)

	got, err := uc.Execute(context.Background(), use_cases.SetCartItemQuantityInput{
		CustomerID: customerID,
		ItemID:     itemID,
		Quantity:   6,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Items()[0].Quantity() != 6 {
		t.Fatalf("quantity = %d, want 6", got.Items()[0].Quantity())
	}
	if repo.getCalls != 1 || repo.saveCalls != 1 {
		t.Fatalf("repository calls: get=%d save=%d, want 1 each", repo.getCalls, repo.saveCalls)
	}
	if len(logger.events) != 1 || logger.events[0].Event != "cart.item_quantity_set" {
		t.Fatalf("unexpected logs: %#v", logger.events)
	}
	if len(metrics.metrics) != 1 || metrics.metrics[0].Labels["result"] != "success" {
		t.Fatalf("unexpected metrics: %#v", metrics.metrics)
	}
}

func TestSetCartItemQuantityRejectsInvalidQuantity(t *testing.T) {
	customerID := uuid.New()
	cart, _ := entities.NewCart(customerID)
	if err := cart.AddItem(uuid.New(), uuid.New(), 2); err != nil {
		t.Fatal(err)
	}

	repo := &cartRepositoryMock{cart: cart}
	uc := use_cases.NewSetCartItemQuantity(repo, &loggerMock{}, &metricsMock{})

	_, err := uc.Execute(context.Background(), use_cases.SetCartItemQuantityInput{
		CustomerID: customerID,
		ItemID:     cart.Items()[0].ID(),
		Quantity:   0,
	})
	if !errors.Is(err, domainerrors.ErrInvalidQuantity) {
		t.Fatalf("Execute() error = %v, want ErrInvalidQuantity", err)
	}
	if repo.saveCalls != 0 {
		t.Fatal("invalid quantity must not be persisted")
	}
}

func TestSetCartItemQuantityRejectsUnknownItem(t *testing.T) {
	customerID := uuid.New()
	cart, _ := entities.NewCart(customerID)
	repo := &cartRepositoryMock{cart: cart}
	uc := use_cases.NewSetCartItemQuantity(repo, &loggerMock{}, &metricsMock{})

	_, err := uc.Execute(context.Background(), use_cases.SetCartItemQuantityInput{
		CustomerID: customerID,
		ItemID:     uuid.New(),
		Quantity:   3,
	})
	if !errors.Is(err, domainerrors.ErrCartItemNotFound) {
		t.Fatalf("Execute() error = %v, want ErrCartItemNotFound", err)
	}
	if repo.saveCalls != 0 {
		t.Fatal("unknown item must not be persisted")
	}
}

func TestSetCartItemQuantityPropagatesSaveFailure(t *testing.T) {
	customerID := uuid.New()
	cart, _ := entities.NewCart(customerID)
	if err := cart.AddItem(uuid.New(), uuid.New(), 1); err != nil {
		t.Fatal(err)
	}
	repositoryErr := errors.New("save failed")
	repo := &cartRepositoryMock{cart: cart, saveErr: repositoryErr}
	logger := &loggerMock{}
	metrics := &metricsMock{}
	uc := use_cases.NewSetCartItemQuantity(repo, logger, metrics)

	_, err := uc.Execute(context.Background(), use_cases.SetCartItemQuantityInput{
		CustomerID: customerID,
		ItemID:     cart.Items()[0].ID(),
		Quantity:   4,
	})
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
