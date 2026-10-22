package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

type cartRepositoryMock struct {
	cart      *entities.Cart
	getErr    error
	saveErr   error
	getCalls  int
	saveCalls int
}

func (m *cartRepositoryMock) GetByCustomerID(
	_ context.Context,
	_ uuid.UUID,
) (*entities.Cart, error) {
	m.getCalls++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.cart == nil {
		return nil, domainerrors.ErrCartNotFound
	}
	return m.cart, nil
}

func (m *cartRepositoryMock) Save(_ context.Context, cart *entities.Cart) error {
	m.saveCalls++
	if m.saveErr != nil {
		return m.saveErr
	}
	m.cart = cart
	return nil
}

func (m *cartRepositoryMock) DeleteByCustomerID(_ context.Context, _ uuid.UUID) error {
	return nil
}

type loggerMock struct {
	events []ports.LogEvent
}

func (m *loggerMock) LogEvent(event ports.LogEvent) {
	m.events = append(m.events, event)
}

type metricsMock struct {
	metrics []ports.Metric
}

func (m *metricsMock) Record(metric ports.Metric) {
	m.metrics = append(m.metrics, metric)
}

func newAddItemUseCase(repo *cartRepositoryMock) (*use_cases.AddItemToCart, *loggerMock, *metricsMock) {
	logger := &loggerMock{}
	metrics := &metricsMock{}
	return use_cases.NewAddItemToCart(repo, logger, metrics), logger, metrics
}

func TestAddItemToCartCreatesCartLazily(t *testing.T) {
	repo := &cartRepositoryMock{}
	uc, logger, metrics := newAddItemUseCase(repo)
	customerID, productID, storeID := uuid.New(), uuid.New(), uuid.New()

	cart, err := uc.Execute(context.Background(), use_cases.AddItemToCartInput{
		CustomerID: customerID,
		ProductID:  productID,
		StoreID:    storeID,
		Quantity:   2,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if cart.CustomerID() != customerID {
		t.Fatalf("CustomerID() = %v, want %v", cart.CustomerID(), customerID)
	}
	if len(cart.Items()) != 1 || cart.Items()[0].Quantity() != 2 {
		t.Fatalf("unexpected cart items: %#v", cart.Items())
	}
	if repo.getCalls != 1 || repo.saveCalls != 1 {
		t.Fatalf("repository calls: get=%d save=%d, want 1 each", repo.getCalls, repo.saveCalls)
	}
	if len(logger.events) != 1 || logger.events[0].Event != "cart.item_added" {
		t.Fatalf("unexpected log events: %#v", logger.events)
	}
	if len(metrics.metrics) != 1 || metrics.metrics[0].Labels["result"] != "success" {
		t.Fatalf("unexpected metrics: %#v", metrics.metrics)
	}
}

func TestAddItemToCartSetsExistingProductQuantity(t *testing.T) {
	customerID, productID, storeID := uuid.New(), uuid.New(), uuid.New()
	cart, _ := entities.NewCart(customerID)
	if err := cart.AddItem(productID, storeID, 2); err != nil {
		t.Fatal(err)
	}

	repo := &cartRepositoryMock{cart: cart}
	uc, _, _ := newAddItemUseCase(repo)

	updated, err := uc.Execute(context.Background(), use_cases.AddItemToCartInput{
		CustomerID: customerID,
		ProductID:  productID,
		StoreID:    storeID,
		Quantity:   5,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := updated.Items()[0].Quantity(); got != 5 {
		t.Fatalf("quantity = %d, want 5", got)
	}
	if len(updated.Items()) != 1 {
		t.Fatalf("items = %d, want 1", len(updated.Items()))
	}
}

func TestAddItemToCartRejectsNilCustomerID(t *testing.T) {
	repo := &cartRepositoryMock{}
	uc, _, _ := newAddItemUseCase(repo)

	_, err := uc.Execute(context.Background(), use_cases.AddItemToCartInput{
		ProductID: uuid.New(),
		StoreID:   uuid.New(),
		Quantity:  1,
	})
	if !errors.Is(err, domainerrors.ErrInvalidCustomerID) {
		t.Fatalf("Execute() error = %v, want ErrInvalidCustomerID", err)
	}
	if repo.getCalls != 0 || repo.saveCalls != 0 {
		t.Fatal("repository should not be called for invalid customer ID")
	}
}

func TestAddItemToCartPropagatesRepositoryFailure(t *testing.T) {
	repositoryErr := errors.New("redis unavailable")
	repo := &cartRepositoryMock{getErr: repositoryErr}
	uc, logger, metrics := newAddItemUseCase(repo)

	_, err := uc.Execute(context.Background(), use_cases.AddItemToCartInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
		StoreID:    uuid.New(),
		Quantity:   1,
	})
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Execute() error = %v, want wrapped repository error", err)
	}
	if repo.saveCalls != 0 {
		t.Fatal("Save() should not be called after GetByCustomerID fails")
	}
	if len(logger.events) != 1 || logger.events[0].FailureCategory != "persistence" {
		t.Fatalf("unexpected failure logs: %#v", logger.events)
	}
	if len(metrics.metrics) != 1 || metrics.metrics[0].Labels["result"] != "failure" {
		t.Fatalf("unexpected failure metrics: %#v", metrics.metrics)
	}
}

func TestAddItemToCartPropagatesSaveFailure(t *testing.T) {
	saveErr := errors.New("save failed")
	repo := &cartRepositoryMock{saveErr: saveErr}
	uc, _, _ := newAddItemUseCase(repo)

	_, err := uc.Execute(context.Background(), use_cases.AddItemToCartInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
		StoreID:    uuid.New(),
		Quantity:   1,
	})
	if !errors.Is(err, saveErr) {
		t.Fatalf("Execute() error = %v, want wrapped save error", err)
	}
}
