package entities

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
)

type PaymentAttempt struct {
	checkoutURL       string
	customerEmail     string
	expiresAt         *time.Time
	metadata          json.RawMessage
	id                string
	paymentID         string
	provider          string
	providerReference string
	status            enums.PaymentAttemptStatus
	failureReason     string
	createdAt         time.Time
	updatedAt         time.Time
}

func NewPaymentAttempt(paymentID, provider string) (*PaymentAttempt, error) {
	paymentID = strings.TrimSpace(paymentID)
	provider = strings.TrimSpace(provider)

	if paymentID == "" || provider == "" {
		return nil, domainerrors.ErrInvalidPaymentAttempt
	}

	id, err := newUUID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &PaymentAttempt{
		id:        id,
		paymentID: paymentID,
		provider:  provider,
		status:    enums.PaymentAttemptStatusPending,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func (a *PaymentAttempt) ID() string {
	return a.id
}

func (a *PaymentAttempt) PaymentID() string {
	return a.paymentID
}

func (a *PaymentAttempt) CustomerEmail() string {
	return a.customerEmail
}

// SetCustomerEmail stores the email used for this checkout attempt.
func (a *PaymentAttempt) SetCustomerEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return domainerrors.ErrInvalidPaymentAttempt
	}

	a.customerEmail = email
	a.touch()
	return nil
}

func (a *PaymentAttempt) Provider() string {
	return a.provider
}

func (a *PaymentAttempt) ProviderReference() string {
	return a.providerReference
}

func (a *PaymentAttempt) Status() enums.PaymentAttemptStatus {
	return a.status
}

func (a *PaymentAttempt) FailureReason() string {
	return a.failureReason
}

func (a *PaymentAttempt) CreatedAt() time.Time {
	return a.createdAt
}

func (a *PaymentAttempt) UpdatedAt() time.Time {
	return a.updatedAt
}

// SetProviderReference records the provider's transaction reference.
// Repeating the same reference is safe; replacing it is rejected.
func (a *PaymentAttempt) SetProviderReference(reference string) error {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return domainerrors.ErrInvalidPaymentAttempt
	}

	if a.providerReference != "" {
		if a.providerReference == reference {
			return nil
		}
		return domainerrors.ErrInvalidPaymentAttempt
	}

	if a.status == enums.PaymentAttemptStatusSucceeded ||
		a.status == enums.PaymentAttemptStatusFailed {
		return domainerrors.ErrInvalidPaymentAttemptTransition
	}

	a.providerReference = reference
	a.touch()
	return nil
}

func (a *PaymentAttempt) MarkProcessing() error {
	if a.status != enums.PaymentAttemptStatusPending {
		return domainerrors.ErrInvalidPaymentAttemptTransition
	}

	a.status = enums.PaymentAttemptStatusProcessing
	a.touch()
	return nil
}

// MarkRequiresReconciliation records an uncertain provider outcome.
// The attempt must be reconciled before its final outcome is recorded.
func (a *PaymentAttempt) MarkRequiresReconciliation() error {
	if a.status != enums.PaymentAttemptStatusProcessing {
		return domainerrors.ErrInvalidPaymentAttemptTransition
	}

	a.status = enums.PaymentAttemptStatusRequiresReconciliation
	a.touch()
	return nil
}

// MarkSucceeded is allowed after processing or after reconciliation
// confirms that the provider completed the charge.
func (a *PaymentAttempt) MarkSucceeded() error {
	if a.status != enums.PaymentAttemptStatusProcessing &&
		a.status != enums.PaymentAttemptStatusRequiresReconciliation {
		return domainerrors.ErrInvalidPaymentAttemptTransition
	}

	a.status = enums.PaymentAttemptStatusSucceeded
	a.failureReason = ""
	a.touch()
	return nil
}

// MarkFailed records a definitive failure, including one confirmed
// by provider reconciliation. An uncertain outcome alone is not failure.
func (a *PaymentAttempt) MarkFailed(reason string) error {
	if a.status != enums.PaymentAttemptStatusProcessing &&
		a.status != enums.PaymentAttemptStatusRequiresReconciliation {
		return domainerrors.ErrInvalidPaymentAttemptTransition
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return domainerrors.ErrInvalidPaymentAttempt
	}

	a.status = enums.PaymentAttemptStatusFailed
	a.failureReason = reason
	a.touch()
	return nil
}

// CheckoutURL returns the provider checkout URL, if one was issued.
func (a *PaymentAttempt) CheckoutURL() string {
	return a.checkoutURL
}

// ExpiresAt returns a copy of the checkout expiry timestamp, if provided.
func (a *PaymentAttempt) ExpiresAt() *time.Time {
	if a.expiresAt == nil {
		return nil
	}
	value := *a.expiresAt
	return &value
}

// Metadata returns a copy of the provider-specific JSON metadata.
func (a *PaymentAttempt) Metadata() json.RawMessage {
	return append(json.RawMessage(nil), a.metadata...)
}

// SetCheckoutDetails stores validated checkout details and provider metadata.
func (a *PaymentAttempt) SetCheckoutDetails(
	checkoutURL string,
	expiresAt *time.Time,
	metadata json.RawMessage,
) error {
	if strings.TrimSpace(checkoutURL) == "" {
		return fmt.Errorf("checkout URL is required")
	}
	if len(metadata) > 0 && !json.Valid(metadata) {
		return fmt.Errorf("provider metadata must be valid JSON")
	}

	a.checkoutURL = checkoutURL
	if expiresAt == nil {
		a.expiresAt = nil
	} else {
		value := *expiresAt
		a.expiresAt = &value
	}
	a.metadata = append(json.RawMessage(nil), metadata...)
	a.touch()
	return nil
}

func (a *PaymentAttempt) touch() {
	a.updatedAt = time.Now().UTC()
}
