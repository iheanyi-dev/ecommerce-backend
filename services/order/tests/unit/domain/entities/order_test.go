package entities_test

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	domain "github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
	entities "github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

func newTestOrder(t *testing.T) (*entities.Order, uuid.UUID, uuid.UUID) {
	t.Helper()

	storeID := uuid.New()
	customerID := uuid.New()

	cartID := uuid.New()
	order, err := entities.NewOrder(
		storeID,
		customerID,
		cartID,
		"12 Market Road, Lagos",
		"NGN",
		30000,
	)
	if err != nil {
		t.Fatalf("entities.NewOrder() error = %v", err)
	}

	return order, storeID, customerID
}

func TestNewOrderCalculatesProductSubtotalAndStartsWithoutShipping(t *testing.T) {
	order, _, _ := newTestOrder(t)

	if order.State != entities.OrderStateCreated {
		t.Fatalf("State = %q, want %q", order.State, entities.OrderStateCreated)
	}
	if order.PriceProduct != 30000 {
		t.Fatalf("PriceProduct = %d, want 30000", order.PriceProduct)
	}
	if order.PriceShipping != 0 {
		t.Fatalf("PriceShipping = %d, want 0", order.PriceShipping)
	}
	if order.Currency != "NGN" {
		t.Fatalf("Currency = %q, want NGN", order.Currency)
	}
	if order.CartID == uuid.Nil {
		t.Fatal("CartID must not be nil")
	}
	if order.PriceProduct != 30000 {
		t.Fatalf("PriceProduct = %d, want 30000", order.PriceProduct)
	}
}

func TestNewOrderRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		storeID    uuid.UUID
		customerID uuid.UUID
		cartID     uuid.UUID
		location   string
		currency   string
		subtotal   int64
		wantErr    error
	}{
		{"missing store", uuid.Nil, uuid.New(), uuid.New(), "Lagos", "NGN", 500, domain.ErrInvalidID},
		{"missing customer", uuid.New(), uuid.Nil, uuid.New(), "Lagos", "NGN", 500, domain.ErrInvalidID},
		{"missing cart", uuid.New(), uuid.New(), uuid.Nil, "Lagos", "NGN", 500, domain.ErrInvalidID},
		{"missing location", uuid.New(), uuid.New(), uuid.New(), " ", "NGN", 500, domain.ErrInvalidDeliveryLocation},
		{"unsupported currency", uuid.New(), uuid.New(), uuid.New(), "Lagos", "XYZ", 500, domain.ErrInvalidCurrency},
		{"negative subtotal", uuid.New(), uuid.New(), uuid.New(), "Lagos", "NGN", -1, domain.ErrInvalidPrice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewOrder(
				tt.storeID, tt.customerID, tt.cartID,
				tt.location, tt.currency, tt.subtotal,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderShippingRejectionAndRevision(t *testing.T) {
	order, storeID, customerID := newTestOrder(t)

	if err := order.AddShipping(storeID, 2500); err != nil {
		t.Fatalf("AddShipping() error = %v", err)
	}
	if order.State != entities.OrderStateShippingAdded {
		t.Fatalf("State = %q, want shipping_added", order.State)
	}

	if err := order.RejectShipping(customerID); err != nil {
		t.Fatalf("RejectShipping() error = %v", err)
	}
	if order.State != entities.OrderStateRejected {
		t.Fatalf("State = %q, want rejected", order.State)
	}

	if err := order.AddShipping(storeID, 3000); err != nil {
		t.Fatalf("revised AddShipping() error = %v", err)
	}
	if order.State != entities.OrderStateShippingAdded {
		t.Fatalf("State = %q, want shipping_added", order.State)
	}
	if order.PriceShipping != 3000 {
		t.Fatalf("PriceShipping = %d, want 3000", order.PriceShipping)
	}
	if order.PriceProduct != 30000 {
		t.Fatalf("product subtotal changed to %d", order.PriceProduct)
	}

	total, err := order.TotalPrice()
	if err != nil {
		t.Fatal(err)
	}
	if total != 33000 {
		t.Fatalf("TotalPrice() = %d, want 33000", total)
	}
}

func TestOrderCancellationRules(t *testing.T) {
	t.Run("store cancels created order", func(t *testing.T) {
		order, storeID, _ := newTestOrder(t)
		if err := order.Cancel(storeID); err != nil {
			t.Fatal(err)
		}
		assertState(t, order, entities.OrderStateCancelled)
	})

	t.Run("customer cannot cancel created order", func(t *testing.T) {
		order, _, customerID := newTestOrder(t)
		if err := order.Cancel(customerID); !errors.Is(err, domain.ErrUnauthorizedActor) {
			t.Fatalf("Cancel() error = %v, want unauthorized actor", err)
		}
	})

	t.Run("customer cancels after shipping proposal", func(t *testing.T) {
		order, storeID, customerID := newTestOrder(t)
		if err := order.AddShipping(storeID, 1000); err != nil {
			t.Fatal(err)
		}
		if err := order.Cancel(customerID); err != nil {
			t.Fatal(err)
		}
		assertState(t, order, entities.OrderStateCancelled)
	})

	t.Run("either party cancels rejected order", func(t *testing.T) {
		for _, actor := range []string{"store", "customer"} {
			t.Run(actor, func(t *testing.T) {
				order, storeID, customerID := newTestOrder(t)
				if err := order.AddShipping(storeID, 1000); err != nil {
					t.Fatal(err)
				}
				if err := order.RejectShipping(customerID); err != nil {
					t.Fatal(err)
				}

				actorID := storeID
				if actor == "customer" {
					actorID = customerID
				}
				if err := order.Cancel(actorID); err != nil {
					t.Fatal(err)
				}
				assertState(t, order, entities.OrderStateCancelled)
			})
		}
	})

	t.Run("cannot cancel payment-successful order", func(t *testing.T) {
		order, storeID, customerID := newTestOrder(t)
		if err := order.AddShipping(storeID, 1000); err != nil {
			t.Fatal(err)
		}
		if err := order.MarkPaymentSuccessful(); err != nil {
			t.Fatal(err)
		}
		if err := order.Cancel(customerID); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("Cancel() error = %v, want invalid transition", err)
		}
		assertState(t, order, entities.OrderStatePaymentSuccessful)
	})

	t.Run("cancelled order is terminal", func(t *testing.T) {
		order, storeID, _ := newTestOrder(t)
		if err := order.Cancel(storeID); err != nil {
			t.Fatal(err)
		}
		if err := order.AddShipping(storeID, 1000); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("AddShipping() error = %v, want invalid transition", err)
		}
	})
}

func TestOrderFulfillmentLifecycle(t *testing.T) {
	order, storeID, customerID := newTestOrder(t)

	if err := order.AddShipping(storeID, 2000); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkPaymentSuccessful(); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkSent(storeID); err != nil {
		t.Fatal(err)
	}
	if err := order.ConfirmReceipt(customerID); err != nil {
		t.Fatal(err)
	}

	assertState(t, order, entities.OrderStateReceived)

	if err := order.MarkTransactionCompleted(); err != nil {
		t.Fatalf("MarkTransactionCompleted() error = %v", err)
	}
	assertState(t, order, entities.OrderStateTransactionCompleted)

	if err := order.MarkSent(storeID); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkSent() after receipt error = %v, want invalid transition", err)
	}
	if err := order.MarkTransactionCompleted(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkTransactionCompleted() twice error = %v, want invalid transition", err)
	}
}

func TestOrderRejectsUnauthorizedActions(t *testing.T) {
	order, storeID, customerID := newTestOrder(t)
	strangerID := uuid.New()

	if err := order.AddShipping(strangerID, 1000); !errors.Is(err, domain.ErrUnauthorizedActor) {
		t.Fatalf("AddShipping() error = %v, want unauthorized actor", err)
	}
	if err := order.RejectShipping(strangerID); !errors.Is(err, domain.ErrUnauthorizedActor) {
		t.Fatalf("RejectShipping() error = %v, want unauthorized actor", err)
	}
	if err := order.MarkSent(storeID); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkSent() error = %v, want invalid transition", err)
	}

	if err := order.AddShipping(storeID, 1000); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkPaymentSuccessful(); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkSent(strangerID); !errors.Is(err, domain.ErrUnauthorizedActor) {
		t.Fatalf("MarkSent() error = %v, want unauthorized actor", err)
	}
	if err := order.ConfirmReceipt(strangerID); !errors.Is(err, domain.ErrUnauthorizedActor) {
		t.Fatalf("ConfirmReceipt() error = %v, want unauthorized actor", err)
	}

	_ = customerID
}

func TestOrderRejectsNegativeShipping(t *testing.T) {
	order, storeID, _ := newTestOrder(t)

	if err := order.AddShipping(storeID, -1); !errors.Is(err, domain.ErrInvalidShippingAmount) {
		t.Fatalf("AddShipping() error = %v, want invalid shipping amount", err)
	}
	if order.State != entities.OrderStateCreated {
		t.Fatalf("State changed after invalid shipping: %q", order.State)
	}
}

func TestOrderRejectsTotalPriceOverflow(t *testing.T) {
	order, _, _ := newTestOrder(t)
	order.PriceProduct = math.MaxInt64
	order.PriceShipping = 1

	if _, err := order.TotalPrice(); !errors.Is(err, domain.ErrPriceOverflow) {
		t.Fatalf("TotalPrice() error = %v, want price overflow", err)
	}
}

func TestNewOrderProductValidation(t *testing.T) {
	tests := []struct {
		name      string
		id        uuid.UUID
		orderID   uuid.UUID
		productID uuid.UUID
		title     string
		qty       int64
		price     int64
		want      error
	}{
		{"missing line ID", uuid.Nil, uuid.New(), uuid.New(), "Mouse", 1, 100, domain.ErrInvalidID},
		{"missing order ID", uuid.New(), uuid.Nil, uuid.New(), "Mouse", 1, 100, domain.ErrInvalidID},
		{"missing product ID", uuid.New(), uuid.New(), uuid.Nil, "Mouse", 1, 100, domain.ErrInvalidID},
		{"empty name", uuid.New(), uuid.New(), uuid.New(), " ", 1, 100, domain.ErrInvalidProduct},
		{"zero quantity", uuid.New(), uuid.New(), uuid.New(), "Mouse", 0, 100, domain.ErrInvalidQuantity},
		{"negative quantity", uuid.New(), uuid.New(), uuid.New(), "Mouse", -1, 100, domain.ErrInvalidQuantity},
		{"negative price", uuid.New(), uuid.New(), uuid.New(), "Mouse", 1, -100, domain.ErrInvalidPrice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewOrderProduct(
				tt.id, tt.orderID, tt.productID, tt.title, tt.qty, tt.price,
			)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func assertState(t *testing.T, order *entities.Order, want entities.OrderState) {
	t.Helper()
	if order.State != want {
		t.Fatalf("State = %q, want %q", order.State, want)
	}
}

func TestOrderPaymentFailureAndRetry(t *testing.T) {
	order, storeID, _ := newTestOrder(t)

	if err := order.AddShipping(storeID, 2000); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkPaymentFailed(); err != nil {
		t.Fatalf("MarkPaymentFailed() error = %v", err)
	}
	assertState(t, order, entities.OrderStatePaymentFailed)

	if err := order.MarkSent(storeID); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkSent() after payment failure error = %v, want invalid transition", err)
	}
	if err := order.MarkTransactionCompleted(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkTransactionCompleted() after payment failure error = %v, want invalid transition", err)
	}

	if err := order.MarkPaymentSuccessful(); err != nil {
		t.Fatalf("MarkPaymentSuccessful() retry error = %v", err)
	}
	assertState(t, order, entities.OrderStatePaymentSuccessful)

	if err := order.MarkPaymentFailed(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkPaymentFailed() after success error = %v, want invalid transition", err)
	}
}

func TestOrderTransactionCannotCompleteBeforeReceipt(t *testing.T) {
	order, storeID, _ := newTestOrder(t)

	if err := order.MarkTransactionCompleted(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkTransactionCompleted() error = %v, want invalid transition", err)
	}

	if err := order.AddShipping(storeID, 1000); err != nil {
		t.Fatal(err)
	}
	if err := order.MarkTransactionCompleted(); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("MarkTransactionCompleted() before payment error = %v, want invalid transition", err)
	}
}
