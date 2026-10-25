package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	domain "github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

// orderLifecycleRepositoryMock simulates loading and updating an order without a database.
type orderLifecycleRepositoryMock struct {
	order       *entities.Order
	findErr     error
	updateErr   error
	findCalls   int
	updateCalls int
}

// CreateMany is unused by lifecycle tests and satisfies the shared repository port.
func (m *orderLifecycleRepositoryMock) CreateMany(
	_ context.Context,
	_ []*entities.Order,
) error {
	return errors.New("CreateMany is not configured in lifecycle tests")
}

// FindByID returns the configured order or configured persistence error.
func (m *orderLifecycleRepositoryMock) FindByID(
	_ context.Context,
	id uuid.UUID,
) (*entities.Order, error) {
	m.findCalls++
	if m.findErr != nil {
		return nil, m.findErr
	}
	if m.order == nil || m.order.ID != id {
		return nil, errors.New("order not found")
	}
	return m.order, nil
}

// Update records an attempted write and simulates a persistence failure when configured.
func (m *orderLifecycleRepositoryMock) Update(
	_ context.Context,
	order *entities.Order,
) error {
	m.updateCalls++
	if m.updateErr != nil {
		return m.updateErr
	}
	m.order = order
	return nil
}

// lifecycleLoggerMock satisfies the application logger port for use-case tests.
type lifecycleLoggerMock struct{}

// Log accepts lifecycle events without introducing external logging dependencies.
func (lifecycleLoggerMock) Log(context.Context, ports.LogEvent) error { return nil }

// lifecycleMetricsMock satisfies the application metrics port for use-case tests.
type lifecycleMetricsMock struct{}

// Increment accepts counter updates without introducing an external metrics dependency.
func (lifecycleMetricsMock) Increment(context.Context, ports.Metric) error { return nil }

// Observe accepts measurement updates without introducing an external metrics dependency.
func (lifecycleMetricsMock) Observe(context.Context, ports.Metric) error { return nil }

// newLifecycleTestOrder creates a valid order for lifecycle tests.
func newLifecycleTestOrder(t *testing.T) *entities.Order {
	t.Helper()

	order, err := entities.NewOrder(
		uuid.New(), uuid.New(), uuid.New(), "Lagos", "NGN", 1000,
	)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	return order
}

// prepareOrderState moves the fixture into the state required by a test.
func prepareOrderState(t *testing.T, order *entities.Order, state entities.OrderState) {
	t.Helper()

	switch state {
	case entities.OrderStateCreated:
		return
	case entities.OrderStateShippingAdded:
		if err := order.AddShipping(order.StoreID, 100); err != nil {
			t.Fatalf("AddShipping() error = %v", err)
		}
	case entities.OrderStateRejected:
		if err := order.AddShipping(order.StoreID, 100); err != nil {
			t.Fatalf("AddShipping() error = %v", err)
		}
		if err := order.RejectShipping(order.CustomerID); err != nil {
			t.Fatalf("RejectShipping() error = %v", err)
		}
	case entities.OrderStatePaid:
		if err := order.AddShipping(order.StoreID, 100); err != nil {
			t.Fatalf("AddShipping() error = %v", err)
		}
		if err := order.MarkPaid(); err != nil {
			t.Fatalf("MarkPaid() error = %v", err)
		}
	case entities.OrderStateSent:
		if err := order.AddShipping(order.StoreID, 100); err != nil {
			t.Fatalf("AddShipping() error = %v", err)
		}
		if err := order.MarkPaid(); err != nil {
			t.Fatalf("MarkPaid() error = %v", err)
		}
		if err := order.MarkSent(order.StoreID); err != nil {
			t.Fatalf("MarkSent() error = %v", err)
		}
	default:
		t.Fatalf("unsupported fixture state %q", state)
	}
}

// TestOrderLifecycleUseCasesApplyValidTransitions verifies every lifecycle use case
// persists a valid domain transition exactly once.
func TestOrderLifecycleUseCasesApplyValidTransitions(t *testing.T) {
	tests := []struct {
		name    string
		initial entities.OrderState
		want    entities.OrderState
		execute func(*orderLifecycleRepositoryMock, *entities.Order) error
	}{
		{
			name: "add shipping", initial: entities.OrderStateCreated,
			want: entities.OrderStateShippingAdded,
			execute: func(repo *orderLifecycleRepositoryMock, order *entities.Order) error {
				return use_cases.NewAddShipping(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
					Execute(context.Background(), use_cases.AddShippingInput{
						OrderID: order.ID, StoreID: order.StoreID, ShippingAmount: 100,
					})
			},
		},
		{
			name: "reject shipping", initial: entities.OrderStateShippingAdded,
			want: entities.OrderStateRejected,
			execute: func(repo *orderLifecycleRepositoryMock, order *entities.Order) error {
				return use_cases.NewRejectShipping(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
					Execute(context.Background(), use_cases.RejectShippingInput{
						OrderID: order.ID, CustomerID: order.CustomerID,
					})
			},
		},
		{
			name: "mark paid", initial: entities.OrderStateShippingAdded,
			want: entities.OrderStatePaid,
			execute: func(repo *orderLifecycleRepositoryMock, order *entities.Order) error {
				return use_cases.NewMarkPaid(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
					Execute(context.Background(), use_cases.MarkPaidInput{OrderID: order.ID})
			},
		},
		{
			name: "cancel order", initial: entities.OrderStateCreated,
			want: entities.OrderStateCancelled,
			execute: func(repo *orderLifecycleRepositoryMock, order *entities.Order) error {
				return use_cases.NewCancelOrder(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
					Execute(context.Background(), use_cases.CancelOrderInput{
						OrderID: order.ID, ActorID: order.StoreID,
					})
			},
		},
		{
			name: "mark sent", initial: entities.OrderStatePaid,
			want: entities.OrderStateSent,
			execute: func(repo *orderLifecycleRepositoryMock, order *entities.Order) error {
				return use_cases.NewMarkSent(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
					Execute(context.Background(), use_cases.MarkSentInput{
						OrderID: order.ID, StoreID: order.StoreID,
					})
			},
		},
		{
			name: "confirm receipt", initial: entities.OrderStateSent,
			want: entities.OrderStateReceived,
			execute: func(repo *orderLifecycleRepositoryMock, order *entities.Order) error {
				return use_cases.NewConfirmReceipt(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
					Execute(context.Background(), use_cases.ConfirmReceiptInput{
						OrderID: order.ID, CustomerID: order.CustomerID,
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := newLifecycleTestOrder(t)
			prepareOrderState(t, order, tt.initial)
			repo := &orderLifecycleRepositoryMock{order: order}

			if err := tt.execute(repo, order); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if repo.findCalls != 1 {
				t.Fatalf("FindByID calls = %d, want 1", repo.findCalls)
			}
			if repo.updateCalls != 1 {
				t.Fatalf("Update calls = %d, want 1", repo.updateCalls)
			}
			if repo.order.State != tt.want {
				t.Fatalf("persisted state = %q, want %q", repo.order.State, tt.want)
			}
		})
	}
}

// TestOrderLifecycleDoesNotPersistInvalidTransitions ensures domain rejections
// never reach the repository's Update method.
func TestOrderLifecycleDoesNotPersistInvalidTransitions(t *testing.T) {
	order := newLifecycleTestOrder(t)
	repo := &orderLifecycleRepositoryMock{order: order}

	err := use_cases.NewMarkPaid(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
		Execute(context.Background(), use_cases.MarkPaidInput{OrderID: order.ID})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Execute() error = %v, want ErrInvalidTransition", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("Update calls = %d, want 0", repo.updateCalls)
	}
	if repo.order.State != entities.OrderStateCreated {
		t.Fatalf("persisted state = %q, want created", repo.order.State)
	}
}

// TestOrderLifecycleRejectsUnauthorizedActor ensures authorization errors from
// the domain are returned without persisting a transition.
func TestOrderLifecycleRejectsUnauthorizedActor(t *testing.T) {
	order := newLifecycleTestOrder(t)
	repo := &orderLifecycleRepositoryMock{order: order}

	err := use_cases.NewAddShipping(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
		Execute(context.Background(), use_cases.AddShippingInput{
			OrderID: order.ID, StoreID: uuid.New(), ShippingAmount: 100,
		})
	if !errors.Is(err, domain.ErrUnauthorizedActor) {
		t.Fatalf("Execute() error = %v, want ErrUnauthorizedActor", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("Update calls = %d, want 0", repo.updateCalls)
	}
}

// TestOrderLifecycleReturnsRepositoryErrors verifies load and update failures
// are propagated and a failed update is not reported as success.
func TestOrderLifecycleReturnsRepositoryErrors(t *testing.T) {
	t.Run("load failure", func(t *testing.T) {
		order := newLifecycleTestOrder(t)
		repo := &orderLifecycleRepositoryMock{order: order, findErr: errors.New("database unavailable")}

		err := use_cases.NewMarkPaid(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
			Execute(context.Background(), use_cases.MarkPaidInput{OrderID: order.ID})
		if err == nil {
			t.Fatal("Execute() error = nil, want load failure")
		}
		if repo.updateCalls != 0 {
			t.Fatalf("Update calls = %d, want 0", repo.updateCalls)
		}
	})

	t.Run("update failure", func(t *testing.T) {
		order := newLifecycleTestOrder(t)
		prepareOrderState(t, order, entities.OrderStateShippingAdded)
		repo := &orderLifecycleRepositoryMock{
			order: order, updateErr: errors.New("database unavailable"),
		}

		err := use_cases.NewMarkPaid(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
			Execute(context.Background(), use_cases.MarkPaidInput{OrderID: order.ID})
		if err == nil {
			t.Fatal("Execute() error = nil, want update failure")
		}
		if repo.updateCalls != 1 {
			t.Fatalf("Update calls = %d, want 1", repo.updateCalls)
		}
	})
}

// TestOrderLifecycleRejectsMissingOrderID verifies invalid input is rejected
// before the repository is called.
func TestOrderLifecycleRejectsMissingOrderID(t *testing.T) {
	repo := &orderLifecycleRepositoryMock{order: newLifecycleTestOrder(t)}

	err := use_cases.NewMarkPaid(repo, lifecycleLoggerMock{}, lifecycleMetricsMock{}).
		Execute(context.Background(), use_cases.MarkPaidInput{OrderID: uuid.Nil})
	if err == nil {
		t.Fatal("Execute() error = nil, want validation error")
	}
	if repo.findCalls != 0 || repo.updateCalls != 0 {
		t.Fatalf("repository calls = find:%d update:%d, want no calls",
			repo.findCalls, repo.updateCalls)
	}
}

// ListByCustomerID is unused by lifecycle tests and returns an empty page.
func (m *orderLifecycleRepositoryMock) ListByCustomerID(
	_ context.Context,
	_ uuid.UUID,
	_, _ int,
) ([]*entities.Order, int64, error) {
	return nil, 0, nil
}

// ListByStoreID is unused by lifecycle tests and returns an empty page.
func (m *orderLifecycleRepositoryMock) ListByStoreID(
	_ context.Context,
	_ uuid.UUID,
	_, _ int,
) ([]*entities.Order, int64, error) {
	return nil, 0, nil
}
