package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/handlers"
)

type paymentHandlerRepo struct {
	order *entities.Order
}

func (m *paymentHandlerRepo) CreateMany(context.Context, []*entities.Order) error {
	return errors.New("not configured")
}
func (m *paymentHandlerRepo) FindByID(_ context.Context, id uuid.UUID) (*entities.Order, error) {
	if m.order == nil || m.order.ID != id {
		return nil, errors.New("order not found")
	}
	return m.order, nil
}
func (m *paymentHandlerRepo) Update(context.Context, *entities.Order) error {
	return errors.New("payment initiation must not update orders")
}
func (m *paymentHandlerRepo) ListByCustomerID(context.Context, uuid.UUID, int, int) ([]*entities.Order, int64, error) {
	return nil, 0, errors.New("not configured")
}
func (m *paymentHandlerRepo) ListByStoreID(context.Context, uuid.UUID, int, int) ([]*entities.Order, int64, error) {
	return nil, 0, errors.New("not configured")
}

type paymentHandlerIdentity struct{}

func (paymentHandlerIdentity) GetUserEmail(context.Context, uuid.UUID) (string, error) {
	return "customer@example.com", nil
}

type paymentHandlerClient struct{}

func (paymentHandlerClient) InitiatePayment(context.Context, ports.InitiatePaymentRequest) (*ports.InitiatePaymentResponse, error) {
	return &ports.InitiatePaymentResponse{
		PaymentID: "payment-id", AttemptID: "attempt-id",
		CheckoutURL: "https://checkout.example/pay",
	}, nil
}
func (paymentHandlerClient) InitiatePayout(context.Context, ports.InitiatePayoutRequest) (*ports.InitiatePayoutResponse, error) {
	return nil, errors.New("not configured")
}

type paymentHandlerLogger struct{}

func (paymentHandlerLogger) Log(context.Context, ports.LogEvent) error { return nil }

type paymentHandlerMetrics struct{}

func (paymentHandlerMetrics) Increment(context.Context, ports.Metric) error { return nil }
func (paymentHandlerMetrics) Observe(context.Context, ports.Metric) error   { return nil }

func newPaymentHandler(t *testing.T) (*handlers.OrderHandlers, uuid.UUID, uuid.UUID) {
	t.Helper()

	customerID := uuid.New()
	order, err := entities.NewOrder(uuid.New(), customerID, uuid.New(), "Lagos", "NGN", 12500)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := order.AddShipping(order.StoreID, 2500); err != nil {
		t.Fatalf("AddShipping() error = %v", err)
	}

	uc := use_cases.NewInitiatePayment(
		&paymentHandlerRepo{order: order},
		paymentHandlerIdentity{},
		paymentHandlerClient{},
		paymentHandlerLogger{},
		paymentHandlerMetrics{},
	)

	return handlers.NewOrderHandlers(nil, nil, nil, nil, nil, nil, nil, nil, uc),
		order.ID, customerID
}

func TestInitiatePaymentHandlerReturnsCheckoutDetails(t *testing.T) {
	h, orderID, customerID := newPaymentHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+orderID.String()+"/payment", nil)
	req.SetPathValue("id", orderID.String())
	req = req.WithContext(ports.WithAuthenticatedIdentity(req.Context(),
		ports.AuthenticatedIdentity{UserID: customerID, Role: "customer"}))

	rec := httptest.NewRecorder()
	h.InitiatePayment(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	for _, expected := range []string{
		`"payment_id":"payment-id"`,
		`"attempt_id":"attempt-id"`,
		`"checkout_url":"https://checkout.example/pay"`,
	} {
		if !contains(rec.Body.String(), expected) {
			t.Fatalf("response %q does not contain %q", rec.Body.String(), expected)
		}
	}
}

func TestInitiatePaymentHandlerRejectsUnauthenticatedRequest(t *testing.T) {
	h, orderID, _ := newPaymentHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+orderID.String()+"/payment", nil)
	req.SetPathValue("id", orderID.String())

	rec := httptest.NewRecorder()
	h.InitiatePayment(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestInitiatePaymentHandlerRejectsWrongMethod(t *testing.T) {
	h, orderID, customerID := newPaymentHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID.String()+"/payment", nil)
	req.SetPathValue("id", orderID.String())
	req = req.WithContext(ports.WithAuthenticatedIdentity(req.Context(),
		ports.AuthenticatedIdentity{UserID: customerID, Role: "customer"}))

	rec := httptest.NewRecorder()
	h.InitiatePayment(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestInitiatePaymentHandlerRejectsInvalidOrderID(t *testing.T) {
	h, _, customerID := newPaymentHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/invalid/payment", nil)
	req.SetPathValue("id", "invalid")
	req = req.WithContext(ports.WithAuthenticatedIdentity(req.Context(),
		ports.AuthenticatedIdentity{UserID: customerID, Role: "customer"}))

	rec := httptest.NewRecorder()
	h.InitiatePayment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func contains(value, part string) bool {
	return len(value) >= len(part) && strings.Contains(value, part)
}
