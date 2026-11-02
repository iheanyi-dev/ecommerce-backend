package ports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

// PaymentProvider abstracts the payment gateway from application logic.
type PaymentProvider interface {
	// InitiateCheckout requests a hosted checkout for a new attempt.
	InitiateCheckout(
		ctx context.Context,
		request CheckoutRequest,
	) (CheckoutResult, error)

	// ReconcileAttempt checks the provider's current result for an existing attempt.
	ReconcileAttempt(
		ctx context.Context,
		request ReconciliationRequest,
	) (ReconciliationResult, error)
}

// CheckoutRequest contains the provider-neutral details required to initiate
// checkout.
type CheckoutRequest struct {
	PaymentID         string
	AttemptID         string
	PaymentReference  string
	CustomerEmail     string
	BusinessReference value_objects.BusinessReference
	Amount            value_objects.Money
}

// CheckoutResult contains normalized checkout details and opaque provider data.
type CheckoutResult struct {
	ProviderReference string
	CheckoutURL       string
	Status            enums.PaymentAttemptStatus
	ExpiresAt         *time.Time
	Metadata          json.RawMessage
}

// ReconciliationRequest identifies an existing provider attempt.
type ReconciliationRequest struct {
	Payment *entities.Payment
	Attempt *entities.PaymentAttempt
}

// ReconciliationResult contains the normalized provider outcome.
type ReconciliationResult struct {
	// Amount and Currency are the provider-verified transaction values.
	Amount            int64
	Currency          string
	ProviderReference string
	CheckoutURL       string
	Status            enums.PaymentAttemptStatus
	FailureReason     string
	Metadata          json.RawMessage
}
