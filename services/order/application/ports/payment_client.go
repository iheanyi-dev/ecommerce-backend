package ports

import (
	"context"

	"github.com/google/uuid"
)

// InitiatePaymentRequest contains the data required to create an order payment.
type InitiatePaymentRequest struct {
	PayerID               uuid.UUID
	Purpose               string
	BusinessReferenceType string
	BusinessReferenceID   uuid.UUID
	AmountMinor           int64
	Currency              string
	CustomerEmail         string
}

// InitiatePaymentResponse contains the Payment identifiers and checkout URL.
type InitiatePaymentResponse struct {
	PaymentID   string
	AttemptID   string
	CheckoutURL string
}

// PayoutBeneficiary contains the bank account receiving the store payout.
type PayoutBeneficiary struct {
	AccountName   string
	AccountNumber string
	BankCode      string
	BankName      string
	CountryCode   string
}

// InitiatePayoutRequest contains the data required to settle an order.
type InitiatePayoutRequest struct {
	BusinessReferenceType string
	BusinessReferenceID   uuid.UUID
	StoreID               uuid.UUID
	OrderID               uuid.UUID
	Reason                string
	AmountMinor           int64
	Currency              string
	Beneficiary           PayoutBeneficiary
}

// InitiatePayoutResponse contains the Payment service payout operation details.
type InitiatePayoutResponse struct {
	PayoutID      string
	AttemptID     string
	Reference     string
	Status        string
	AttemptStatus string
	Provider      string
	AmountMinor   int64
	Currency      string
}

// PaymentClient initiates payments and payouts through the Payment service.
type PaymentClient interface {
	InitiatePayment(
		ctx context.Context,
		request InitiatePaymentRequest,
	) (*InitiatePaymentResponse, error)
}
