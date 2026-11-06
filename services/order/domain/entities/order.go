package entities

import (
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/iheanyi-dev/ecommerce-backend/services/order/domain"
)

type OrderState string

const (
	OrderStateCreated              OrderState = "created"
	OrderStateShippingAdded        OrderState = "shipping_added"
	OrderStatePaymentSuccessful    OrderState = "payment_successful"
	OrderStatePaymentFailed        OrderState = "payment_failed"
	OrderStateRejected             OrderState = "rejected"
	OrderStateSent                 OrderState = "sent"
	OrderStateReceived             OrderState = "received"
	OrderStateTransactionCompleted OrderState = "transaction_completed"
	OrderStateCancelled            OrderState = "cancelled"
)

// Order represents one customer's order from one store.
// CartID groups orders created during the same checkout.
type Order struct {
	ID               uuid.UUID
	CartID           uuid.UUID
	StoreID          uuid.UUID
	CustomerID       uuid.UUID
	DeliveryLocation string
	PriceProduct     int64
	PriceShipping    int64
	Currency         string
	State            OrderState
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewOrder creates an order without embedding its product lines.
// The application layer calculates PriceProduct from validated snapshots.
func NewOrder(
	storeID uuid.UUID,
	customerID uuid.UUID,
	cartID uuid.UUID,
	deliveryLocation string,
	currency string,
	priceProduct int64,
) (*Order, error) {
	if storeID == uuid.Nil || customerID == uuid.Nil || cartID == uuid.Nil {
		return nil, domain.ErrInvalidID
	}
	if strings.TrimSpace(deliveryLocation) == "" {
		return nil, domain.ErrInvalidDeliveryLocation
	}
	if !validCurrency(currency) {
		return nil, domain.ErrInvalidCurrency
	}
	if priceProduct < 0 {
		return nil, domain.ErrInvalidPrice
	}

	now := time.Now().UTC()

	return &Order{
		ID:               uuid.New(),
		CartID:           cartID,
		StoreID:          storeID,
		CustomerID:       customerID,
		DeliveryLocation: strings.TrimSpace(deliveryLocation),
		PriceProduct:     priceProduct,
		Currency:         strings.ToUpper(strings.TrimSpace(currency)),
		State:            OrderStateCreated,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// AddShipping lets the owning store propose or revise shipping.
func (o *Order) AddShipping(storeID uuid.UUID, amount int64) error {
	if storeID != o.StoreID {
		return domain.ErrUnauthorizedActor
	}
	if o.State != OrderStateCreated && o.State != OrderStateRejected {
		return domain.ErrInvalidTransition
	}
	if amount < 0 {
		return domain.ErrInvalidShippingAmount
	}

	o.PriceShipping = amount
	o.State = OrderStateShippingAdded
	o.touch()
	return nil
}

// RejectShipping records the customer's rejection of the current proposal.
func (o *Order) RejectShipping(customerID uuid.UUID) error {
	if customerID != o.CustomerID {
		return domain.ErrUnauthorizedActor
	}
	if o.State != OrderStateShippingAdded {
		return domain.ErrInvalidTransition
	}

	o.State = OrderStateRejected
	o.touch()
	return nil
}

// MarkPaymentSuccessful records trusted confirmation that payment succeeded.
// A failed payment can be retried without creating another order.
func (o *Order) MarkPaymentSuccessful() error {
	if o.State != OrderStateShippingAdded && o.State != OrderStatePaymentFailed {
		return domain.ErrInvalidTransition
	}

	o.State = OrderStatePaymentSuccessful
	o.touch()
	return nil
}

// MarkPaymentFailed records trusted confirmation that payment failed.
func (o *Order) MarkPaymentFailed() error {
	if o.State != OrderStateShippingAdded {
		return domain.ErrInvalidTransition
	}

	o.State = OrderStatePaymentFailed
	o.touch()
	return nil
}

// Cancel terminates the order according to its state and actor.
func (o *Order) Cancel(actorID uuid.UUID) error {
	if actorID == uuid.Nil {
		return domain.ErrUnauthorizedActor
	}

	switch o.State {
	case OrderStateCreated:
		if actorID != o.StoreID {
			return domain.ErrUnauthorizedActor
		}
	case OrderStateShippingAdded:
		if actorID != o.CustomerID {
			return domain.ErrUnauthorizedActor
		}
	case OrderStateRejected:
		if actorID != o.StoreID && actorID != o.CustomerID {
			return domain.ErrUnauthorizedActor
		}
	default:
		return domain.ErrInvalidTransition
	}

	o.State = OrderStateCancelled
	o.touch()
	return nil
}

// MarkSent records dispatch by the owning store.
func (o *Order) MarkSent(storeID uuid.UUID) error {
	if storeID != o.StoreID {
		return domain.ErrUnauthorizedActor
	}
	if o.State != OrderStatePaymentSuccessful {
		return domain.ErrInvalidTransition
	}

	o.State = OrderStateSent
	o.touch()
	return nil
}

// ConfirmReceipt records receipt confirmation by the owning customer.
func (o *Order) ConfirmReceipt(customerID uuid.UUID) error {
	if customerID != o.CustomerID {
		return domain.ErrUnauthorizedActor
	}
	if o.State != OrderStateSent {
		return domain.ErrInvalidTransition
	}

	o.State = OrderStateReceived
	o.touch()
	return nil
}

// MarkTransactionCompleted records trusted confirmation that the store payout
// succeeded after the customer confirmed receipt.
func (o *Order) MarkTransactionCompleted() error {
	if o.State != OrderStateReceived {
		return domain.ErrInvalidTransition
	}

	o.State = OrderStateTransactionCompleted
	o.touch()
	return nil
}

// TotalPrice returns the product subtotal plus shipping.
func (o *Order) TotalPrice() (int64, error) {
	if o.PriceShipping < 0 || o.PriceProduct < 0 {
		return 0, domain.ErrInvalidPrice
	}
	if o.PriceProduct > math.MaxInt64-o.PriceShipping {
		return 0, domain.ErrPriceOverflow
	}
	return o.PriceProduct + o.PriceShipping, nil
}

func (o *Order) touch() {
	o.UpdatedAt = time.Now().UTC()
}

func validCurrency(currency string) bool {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "NGN", "USD", "EUR", "GBP", "JPY",
		"KES", "GHS", "ZAR", "CAD", "AUD",
		"INR", "CNY":
		return true
	default:
		return false
	}
}
