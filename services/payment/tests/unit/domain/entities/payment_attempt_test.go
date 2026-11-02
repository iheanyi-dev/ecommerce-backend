package entities_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
)

func newTestPaymentAttempt(t *testing.T) *entities.PaymentAttempt {
	t.Helper()

	attempt, err := entities.NewPaymentAttempt("payment-123", " Paystack ")
	if err != nil {
		t.Fatalf("NewPaymentAttempt() error = %v", err)
	}
	return attempt
}

func TestNewPaymentAttempt(t *testing.T) {
	attempt := newTestPaymentAttempt(t)

	if attempt.ID() == "" {
		t.Error("ID must not be empty")
	}
	if attempt.PaymentID() != "payment-123" {
		t.Errorf("PaymentID() = %q", attempt.PaymentID())
	}
	if attempt.Provider() != "Paystack" {
		t.Errorf("Provider() = %q, want Paystack", attempt.Provider())
	}
	if attempt.Status() != enums.PaymentAttemptStatusPending {
		t.Errorf("Status() = %q, want pending", attempt.Status())
	}
	if attempt.CreatedAt().IsZero() || attempt.UpdatedAt().IsZero() {
		t.Error("timestamps must be initialized")
	}
	if !attempt.CreatedAt().Equal(attempt.UpdatedAt()) {
		t.Error("initial timestamps should be equal")
	}
	if attempt.ProviderReference() != "" || attempt.FailureReason() != "" {
		t.Error("provider reference and failure reason should initially be empty")
	}
}

func TestNewPaymentAttemptRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		paymentID string
		provider  string
	}{
		{name: "empty payment ID", paymentID: "", provider: "Paystack"},
		{name: "whitespace payment ID", paymentID: "   ", provider: "Paystack"},
		{name: "empty provider", paymentID: "payment-123", provider: ""},
		{name: "whitespace provider", paymentID: "payment-123", provider: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewPaymentAttempt(tt.paymentID, tt.provider)
			if !errors.Is(err, domainerrors.ErrInvalidPaymentAttempt) {
				t.Errorf("error = %v, want ErrInvalidPaymentAttempt", err)
			}
		})
	}
}

func TestPaymentAttemptProviderReference(t *testing.T) {
	attempt := newTestPaymentAttempt(t)

	if err := attempt.SetProviderReference("  provider-tx-123  "); err != nil {
		t.Fatal(err)
	}
	if attempt.ProviderReference() != "provider-tx-123" {
		t.Errorf("ProviderReference() = %q", attempt.ProviderReference())
	}

	updatedAt := attempt.UpdatedAt()
	if err := attempt.SetProviderReference("provider-tx-123"); err != nil {
		t.Fatalf("setting the same reference should be idempotent: %v", err)
	}
	if !attempt.UpdatedAt().Equal(updatedAt) {
		t.Error("idempotent reference assignment changed UpdatedAt")
	}

	if err := attempt.SetProviderReference("different-reference"); !errors.Is(err, domainerrors.ErrInvalidPaymentAttempt) {
		t.Errorf("error = %v, want ErrInvalidPaymentAttempt", err)
	}
	if attempt.ProviderReference() != "provider-tx-123" {
		t.Error("existing provider reference was overwritten")
	}
	if err := attempt.SetProviderReference(" "); !errors.Is(err, domainerrors.ErrInvalidPaymentAttempt) {
		t.Errorf("empty reference error = %v", err)
	}
}

func TestPaymentAttemptLifecycle(t *testing.T) {
	attempt := newTestPaymentAttempt(t)

	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkRequiresReconciliation(); err != nil {
		t.Fatal(err)
	}
	if attempt.Status() != enums.PaymentAttemptStatusRequiresReconciliation {
		t.Fatalf("Status() = %q", attempt.Status())
	}

	// Reconciliation confirms that the provider completed the charge.
	if err := attempt.MarkSucceeded(); err != nil {
		t.Fatal(err)
	}
	if attempt.Status() != enums.PaymentAttemptStatusSucceeded {
		t.Errorf("Status() = %q, want succeeded", attempt.Status())
	}
	if attempt.FailureReason() != "" {
		t.Error("successful attempt must not have a failure reason")
	}
	if err := attempt.MarkFailed("late failure"); !errors.Is(err, domainerrors.ErrInvalidPaymentAttemptTransition) {
		t.Errorf("terminal transition error = %v", err)
	}
}

func TestPaymentAttemptDefinitiveFailure(t *testing.T) {
	attempt := newTestPaymentAttempt(t)

	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkFailed("  card declined  "); err != nil {
		t.Fatal(err)
	}

	if attempt.Status() != enums.PaymentAttemptStatusFailed {
		t.Errorf("Status() = %q, want failed", attempt.Status())
	}
	if attempt.FailureReason() != "card declined" {
		t.Errorf("FailureReason() = %q", attempt.FailureReason())
	}
}

func TestPaymentAttemptRejectsFailureWithoutReason(t *testing.T) {
	attempt := newTestPaymentAttempt(t)
	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}

	if err := attempt.MarkFailed(" "); !errors.Is(err, domainerrors.ErrInvalidPaymentAttempt) {
		t.Errorf("error = %v, want ErrInvalidPaymentAttempt", err)
	}
	if attempt.Status() != enums.PaymentAttemptStatusProcessing {
		t.Error("invalid failure request changed status")
	}
}

func TestPaymentAttemptCannotBeFailedDirectlyWhilePending(t *testing.T) {
	attempt := newTestPaymentAttempt(t)
	before := attempt.UpdatedAt()

	if err := attempt.MarkFailed("declined"); !errors.Is(err, domainerrors.ErrInvalidPaymentAttemptTransition) {
		t.Errorf("error = %v, want ErrInvalidPaymentAttemptTransition", err)
	}
	if attempt.Status() != enums.PaymentAttemptStatusPending {
		t.Error("rejected transition changed status")
	}
	if !attempt.UpdatedAt().Equal(before) {
		t.Error("rejected transition changed UpdatedAt")
	}
}

func TestPaymentAttemptUnknownOutcomeMustBeResolvedExplicitly(t *testing.T) {
	attempt := newTestPaymentAttempt(t)
	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkRequiresReconciliation(); err != nil {
		t.Fatal(err)
	}

	before := attempt.UpdatedAt()
	if err := attempt.MarkProcessing(); !errors.Is(err, domainerrors.ErrInvalidPaymentAttemptTransition) {
		t.Errorf("error = %v, want ErrInvalidPaymentAttemptTransition", err)
	}
	if err := attempt.MarkRequiresReconciliation(); !errors.Is(err, domainerrors.ErrInvalidPaymentAttemptTransition) {
		t.Errorf("error = %v, want ErrInvalidPaymentAttemptTransition", err)
	}
	if !attempt.UpdatedAt().Equal(before) {
		t.Error("rejected transition changed UpdatedAt")
	}
	if attempt.Status() != enums.PaymentAttemptStatusRequiresReconciliation {
		t.Error("unknown outcome was not preserved")
	}

	// Only a definitive reconciliation result can finalize this attempt.
	if err := attempt.MarkFailed("provider confirmed no charge"); err != nil {
		t.Fatal(err)
	}
	if attempt.Status() != enums.PaymentAttemptStatusFailed {
		t.Errorf("Status() = %q, want failed", attempt.Status())
	}
}

func TestPaymentAttemptCannotChangeProviderReferenceAfterCompletion(t *testing.T) {
	attempt := newTestPaymentAttempt(t)
	if err := attempt.SetProviderReference("provider-tx-123"); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkSucceeded(); err != nil {
		t.Fatal(err)
	}

	if err := attempt.SetProviderReference("provider-tx-123"); err != nil {
		t.Errorf("replaying the same reference after completion should be idempotent: %v", err)
	}
	if err := attempt.SetProviderReference("provider-tx-456"); !errors.Is(err, domainerrors.ErrInvalidPaymentAttempt) {
		t.Errorf("error = %v, want ErrInvalidPaymentAttemptTransition", err)
	}
	if attempt.ProviderReference() != "provider-tx-123" {
		t.Error("provider reference changed after completion")
	}
}

func TestPaymentAttemptCustomerEmail(t *testing.T) {
	attempt := newTestPaymentAttempt(t)

	if err := attempt.SetCustomerEmail("  customer@example.com  "); err != nil {
		t.Fatal(err)
	}

	if got := attempt.CustomerEmail(); got != "customer@example.com" {
		t.Errorf("CustomerEmail() = %q, want customer@example.com", got)
	}
}

func TestPaymentAttemptRejectsEmptyCustomerEmail(t *testing.T) {
	attempt := newTestPaymentAttempt(t)

	if err := attempt.SetCustomerEmail(" "); !errors.Is(err, domainerrors.ErrInvalidPaymentAttempt) {
		t.Errorf("error = %v, want ErrInvalidPaymentAttempt", err)
	}

	if got := attempt.CustomerEmail(); got != "" {
		t.Errorf("CustomerEmail() = %q, want empty", got)
	}
}
