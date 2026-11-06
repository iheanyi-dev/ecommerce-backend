package use_cases_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

type initiatePaymentOrderRepository struct {
	order *entities.Order
	err   error
	calls int
}

func (m *initiatePaymentOrderRepository) CreateMany(context.Context, []*entities.Order) error {
	return errors.New("CreateMany is not configured")
}

func (m *initiatePaymentOrderRepository) FindByID(_ context.Context, id uuid.UUID) (*entities.Order, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	if m.order == nil || m.order.ID != id {
		return nil, errors.New("order not found")
	}
	return m.order, nil
}

func (m *initiatePaymentOrderRepository) Update(context.Context, *entities.Order) error {
	return errors.New("InitiatePayment must not update the order")
}

func (m *initiatePaymentOrderRepository) ListByCustomerID(context.Context, uuid.UUID, int, int) ([]*entities.Order, int64, error) {
	return nil, 0, errors.New("ListByCustomerID is not configured")
}

func (m *initiatePaymentOrderRepository) ListByStoreID(context.Context, uuid.UUID, int, int) ([]*entities.Order, int64, error) {
	return nil, 0, errors.New("ListByStoreID is not configured")
}

type initiatePaymentIdentityMock struct {
	email  string
	err    error
	userID uuid.UUID
	calls  int
}

func (m *initiatePaymentIdentityMock) GetUserEmail(_ context.Context, userID uuid.UUID) (string, error) {
	m.calls++
	m.userID = userID
	return m.email, m.err
}

type initiatePaymentClientMock struct {
	request  ports.InitiatePaymentRequest
	response *ports.InitiatePaymentResponse
	err      error
	calls    int
}

func (m *initiatePaymentClientMock) InitiatePayment(
	_ context.Context,
	request ports.InitiatePaymentRequest,
) (*ports.InitiatePaymentResponse, error) {
	m.calls++
	m.request = request
	return m.response, m.err
}

func (m *initiatePaymentClientMock) InitiatePayout(
	_ context.Context,
	_ ports.InitiatePayoutRequest,
) (*ports.InitiatePayoutResponse, error) {
	return nil, errors.New("InitiatePayout is not configured")
}

type initiatePaymentLoggerMock struct{}

func (initiatePaymentLoggerMock) Log(context.Context, ports.LogEvent) error { return nil }

type initiatePaymentMetricsMock struct{}

func (initiatePaymentMetricsMock) Increment(context.Context, ports.Metric) error { return nil }
func (initiatePaymentMetricsMock) Observe(context.Context, ports.Metric) error   { return nil }

func newInitiatePaymentOrder(t *testing.T, state entities.OrderState) *entities.Order {
	t.Helper()

	order, err := entities.NewOrder(uuid.New(), uuid.New(), uuid.New(), "Lagos", "NGN", 12500)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := order.AddShipping(order.StoreID, 2500); err != nil {
		t.Fatalf("AddShipping() error = %v", err)
	}
	if state == entities.OrderStatePaymentFailed {
		if err := order.MarkPaymentFailed(); err != nil {
			t.Fatalf("MarkPaymentFailed() error = %v", err)
		}
	}
	return order
}

func newInitiatePaymentUseCase(
	repo *initiatePaymentOrderRepository,
	identity *initiatePaymentIdentityMock,
	payment *initiatePaymentClientMock,
) *use_cases.InitiatePayment {
	return use_cases.NewInitiatePayment(
		repo,
		identity,
		payment,
		initiatePaymentLoggerMock{},
		initiatePaymentMetricsMock{},
	)
}

func TestInitiatePaymentSendsOrderTotalAndReturnsCheckoutURL(t *testing.T) {
	for _, state := range []entities.OrderState{
		entities.OrderStateShippingAdded,
		entities.OrderStatePaymentFailed,
	} {
		t.Run(string(state), func(t *testing.T) {
			order := newInitiatePaymentOrder(t, state)
			repo := &initiatePaymentOrderRepository{order: order}
			identity := &initiatePaymentIdentityMock{email: " customer@example.com "}
			payment := &initiatePaymentClientMock{
				response: &ports.InitiatePaymentResponse{
					PaymentID:   "payment-id",
					AttemptID:   "attempt-id",
					CheckoutURL: "https://checkout.example/pay",
				},
			}

			uc := newInitiatePaymentUseCase(repo, identity, payment)

			result, err := uc.Execute(context.Background(), use_cases.InitiatePaymentInput{
				OrderID:    order.ID,
				CustomerID: order.CustomerID,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if result.CheckoutURL != payment.response.CheckoutURL ||
				result.PaymentID != payment.response.PaymentID ||
				result.AttemptID != payment.response.AttemptID {
				t.Fatalf("unexpected result: %+v", result)
			}
			if identity.calls != 1 || identity.userID != order.CustomerID {
				t.Fatalf("Identity lookup calls = %d, user ID = %s", identity.calls, identity.userID)
			}
			if payment.calls != 1 {
				t.Fatalf("Payment calls = %d, want 1", payment.calls)
			}

			request := payment.request
			if request.PayerID != order.CustomerID ||
				request.Purpose != "order" ||
				request.BusinessReferenceType != "order" ||
				request.BusinessReferenceID != order.ID ||
				request.AmountMinor != 15000 ||
				request.Currency != "NGN" ||
				request.CustomerEmail != "customer@example.com" {
				t.Fatalf("unexpected Payment request: %+v", request)
			}
			if repo.calls != 1 {
				t.Fatalf("FindByID calls = %d, want 1", repo.calls)
			}
			if order.State != state {
				t.Fatalf("order state changed to %q, want %q", order.State, state)
			}
		})
	}
}

func TestInitiatePaymentRejectsInvalidInputOwnershipAndOrderState(t *testing.T) {
	tests := []struct {
		name       string
		orderID    uuid.UUID
		customerID uuid.UUID
		state      entities.OrderState
	}{
		{name: "missing order ID", customerID: uuid.New(), state: entities.OrderStateShippingAdded},
		{name: "missing customer ID", orderID: uuid.New(), state: entities.OrderStateShippingAdded},
		{name: "customer does not own order", orderID: uuid.New(), customerID: uuid.New(), state: entities.OrderStateShippingAdded},
		{name: "created order", orderID: uuid.New(), state: entities.OrderStateCreated},
		{name: "rejected order", orderID: uuid.New(), state: entities.OrderStateRejected},
		{name: "already paid", orderID: uuid.New(), state: entities.OrderStatePaymentSuccessful},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := newInitiatePaymentOrder(t, entities.OrderStateShippingAdded)
			order.ID = tt.orderID
			order.State = tt.state
			if tt.name == "customer does not own order" {
				order.CustomerID = uuid.New()
			}

			customerID := tt.customerID
			if tt.name == "created order" || tt.name == "rejected order" || tt.name == "already paid" {
				customerID = order.CustomerID
			}

			repo := &initiatePaymentOrderRepository{order: order}
			identity := &initiatePaymentIdentityMock{email: "customer@example.com"}
			payment := &initiatePaymentClientMock{
				response: &ports.InitiatePaymentResponse{CheckoutURL: "https://checkout.example/pay"},
			}

			uc := newInitiatePaymentUseCase(repo, identity, payment)
			_, err := uc.Execute(context.Background(), use_cases.InitiatePaymentInput{
				OrderID:    tt.orderID,
				CustomerID: customerID,
			})
			if err == nil {
				t.Fatal("Execute() error = nil, want error")
			}
			if identity.calls != 0 {
				t.Fatalf("Identity calls = %d, want 0", identity.calls)
			}
			if payment.calls != 0 {
				t.Fatalf("Payment calls = %d, want 0", payment.calls)
			}
		})
	}
}

func TestInitiatePaymentPropagatesIdentityError(t *testing.T) {
	identityErr := errors.New("Identity service unavailable")
	order := newInitiatePaymentOrder(t, entities.OrderStateShippingAdded)
	repo := &initiatePaymentOrderRepository{order: order}
	identity := &initiatePaymentIdentityMock{err: identityErr}
	payment := &initiatePaymentClientMock{}

	uc := newInitiatePaymentUseCase(repo, identity, payment)
	_, err := uc.Execute(context.Background(), use_cases.InitiatePaymentInput{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
	})
	if err == nil || !strings.Contains(err.Error(), identityErr.Error()) {
		t.Fatalf("Execute() error = %v, want containing %q", err, identityErr)
	}
	if payment.calls != 0 {
		t.Fatalf("Payment calls = %d, want 0", payment.calls)
	}
}

func TestInitiatePaymentRejectsEmptyIdentityEmail(t *testing.T) {
	order := newInitiatePaymentOrder(t, entities.OrderStateShippingAdded)
	identity := &initiatePaymentIdentityMock{email: "  "}
	payment := &initiatePaymentClientMock{}

	uc := newInitiatePaymentUseCase(
		&initiatePaymentOrderRepository{order: order},
		identity,
		payment,
	)
	_, err := uc.Execute(context.Background(), use_cases.InitiatePaymentInput{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if payment.calls != 0 {
		t.Fatalf("Payment calls = %d, want 0", payment.calls)
	}
}

func TestInitiatePaymentPropagatesPaymentErrorAndRejectsMissingCheckoutURL(t *testing.T) {
	paymentErr := errors.New("Payment service unavailable")
	tests := []struct {
		name    string
		client  *initiatePaymentClientMock
		wantErr string
	}{
		{
			name:    "payment service error",
			client:  &initiatePaymentClientMock{err: paymentErr},
			wantErr: "Payment service unavailable",
		},
		{
			name:    "nil response",
			client:  &initiatePaymentClientMock{},
			wantErr: "no checkout URL",
		},
		{
			name:    "empty checkout URL",
			client:  &initiatePaymentClientMock{response: &ports.InitiatePaymentResponse{}},
			wantErr: "no checkout URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := newInitiatePaymentOrder(t, entities.OrderStateShippingAdded)
			identity := &initiatePaymentIdentityMock{email: "customer@example.com"}
			uc := newInitiatePaymentUseCase(
				&initiatePaymentOrderRepository{order: order},
				identity,
				tt.client,
			)

			_, err := uc.Execute(context.Background(), use_cases.InitiatePaymentInput{
				OrderID:    order.ID,
				CustomerID: order.CustomerID,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Execute() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
