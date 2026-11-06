package ports

import "context"

const (
	// PaymentPurposeSubscription identifies a store subscription payment.
	PaymentPurposeSubscription = "subscription"

	// BusinessReferenceTypeSubscription identifies the Billing subscription
	// whose payment is being initiated.
	BusinessReferenceTypeSubscription = "subscription"
)

// InitiatePaymentRequest contains the information Billing sends to Payment.
type InitiatePaymentRequest struct {
	PayerID               string
	Purpose               string
	BusinessReferenceType string
	BusinessReferenceID   string
	AmountMinor           int64
	Currency              string
	CustomerEmail         string
}

// InitiatePaymentResult contains the checkout details returned by Payment.
type InitiatePaymentResult struct {
	PaymentID     string
	AttemptID     string
	PaymentStatus string
	AttemptStatus string
	Provider      string
	CheckoutURL   string
	AmountMinor   int64
	Currency      string
}

// Payment defines the application contract for initiating subscription payments.
// The infrastructure adapter is responsible for translating this contract
// to and from the Payment service's generated gRPC messages.
type Payment interface {
	InitiatePayment(
		ctx context.Context,
		request InitiatePaymentRequest,
	) (InitiatePaymentResult, error)
}
