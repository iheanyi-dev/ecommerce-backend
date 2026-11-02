package entities_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
)

func newTestPayoutAttempt(t *testing.T) *entities.PayoutAttempt {
	t.Helper()

	attempt, err := entities.NewPayoutAttempt("payout-123", "Paystack")
	if err != nil {
		t.Fatal(err)
	}
	return attempt
}

func TestNewPayoutAttempt(t *testing.T) {
	attempt := newTestPayoutAttempt(t)

	if attempt.ID() == "" {
		t.Error("attempt ID must not be empty")
	}
	if attempt.PayoutID() != "payout-123" {
		t.Errorf("PayoutID() = %q", attempt.PayoutID())
	}
	if attempt.Provider() != "Paystack" {
		t.Errorf("Provider() = %q", attempt.Provider())
	}
	if attempt.Status() != enums.PayoutAttemptStatusPending {
		t.Errorf("Status() = %q, want pending", attempt.Status())
	}
	if attempt.CreatedAt().IsZero() || !attempt.CreatedAt().Equal(attempt.UpdatedAt()) {
		t.Error("initial timestamps should be set and equal")
	}
}

func TestNewPayoutAttemptRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		payout   string
		provider string
	}{
		{name: "empty payout ID", payout: "", provider: "Paystack"},
		{name: "empty provider", payout: "payout-1", provider: ""},
		{name: "whitespace provider", payout: "payout-1", provider: "  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewPayoutAttempt(tt.payout, tt.provider)
			if !errors.Is(err, domainerrors.ErrInvalidPayoutAttempt) {
				t.Errorf("error = %v, want ErrInvalidPayoutAttempt", err)
			}
		})
	}
}

func TestPayoutAttemptProviderReference(t *testing.T) {
	attempt := newTestPayoutAttempt(t)

	if err := attempt.SetProviderReference(" provider-transfer-123 "); err != nil {
		t.Fatal(err)
	}
	if attempt.ProviderReference() != "provider-transfer-123" {
		t.Errorf("ProviderReference() = %q", attempt.ProviderReference())
	}
	if err := attempt.SetProviderReference("provider-transfer-123"); err != nil {
		t.Fatalf("same reference should be idempotent: %v", err)
	}
	if err := attempt.SetProviderReference("different-reference"); !errors.Is(err, domainerrors.ErrInvalidPayoutAttempt) {
		t.Errorf("error = %v, want ErrInvalidPayoutAttempt", err)
	}
}

func TestPayoutAttemptLifecycle(t *testing.T) {
	attempt := newTestPayoutAttempt(t)

	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkRequiresReconciliation(); err != nil {
		t.Fatal(err)
	}
	if attempt.Status() != enums.PayoutAttemptStatusRequiresReconciliation {
		t.Fatalf("Status() = %q", attempt.Status())
	}
	if err := attempt.MarkSucceeded(); err != nil {
		t.Fatal(err)
	}
	if attempt.Status() != enums.PayoutAttemptStatusSucceeded {
		t.Errorf("Status() = %q, want succeeded", attempt.Status())
	}
	if err := attempt.MarkFailed("late failure"); !errors.Is(err, domainerrors.ErrInvalidPayoutAttemptTransition) {
		t.Errorf("terminal transition error = %v", err)
	}
}

func TestPayoutAttemptDefinitiveFailure(t *testing.T) {
	attempt := newTestPayoutAttempt(t)

	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkFailed("  transfer rejected  "); err != nil {
		t.Fatal(err)
	}
	if attempt.Status() != enums.PayoutAttemptStatusFailed {
		t.Errorf("Status() = %q, want failed", attempt.Status())
	}
	if attempt.FailureReason() != "transfer rejected" {
		t.Errorf("FailureReason() = %q", attempt.FailureReason())
	}
}

func TestPayoutAttemptRejectsInvalidTransitionsWithoutMutation(t *testing.T) {
	attempt := newTestPayoutAttempt(t)
	before := attempt.UpdatedAt()

	if err := attempt.MarkSucceeded(); !errors.Is(err, domainerrors.ErrInvalidPayoutAttemptTransition) {
		t.Fatalf("error = %v", err)
	}
	if err := attempt.MarkFailed("declined"); !errors.Is(err, domainerrors.ErrInvalidPayoutAttemptTransition) {
		t.Fatalf("error = %v", err)
	}
	if attempt.Status() != enums.PayoutAttemptStatusPending {
		t.Error("rejected transition changed attempt status")
	}
	if !attempt.UpdatedAt().Equal(before) {
		t.Error("rejected transition changed UpdatedAt")
	}
}

func TestPayoutAttemptRequiresFailureReason(t *testing.T) {
	attempt := newTestPayoutAttempt(t)
	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}

	if err := attempt.MarkFailed(" "); !errors.Is(err, domainerrors.ErrInvalidPayoutAttempt) {
		t.Errorf("error = %v, want ErrInvalidPayoutAttempt", err)
	}
	if attempt.Status() != enums.PayoutAttemptStatusProcessing {
		t.Error("invalid failure request changed attempt status")
	}
}

func TestPayoutAttemptCannotChangeReferenceAfterCompletion(t *testing.T) {
	attempt := newTestPayoutAttempt(t)
	if err := attempt.SetProviderReference("transfer-123"); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := attempt.MarkSucceeded(); err != nil {
		t.Fatal(err)
	}

	if err := attempt.SetProviderReference("transfer-123"); err != nil {
		t.Errorf("replaying the same reference after completion should be idempotent: %v", err)
	}
	if err := attempt.SetProviderReference("transfer-456"); !errors.Is(err, domainerrors.ErrInvalidPayoutAttempt) {
		t.Errorf("error = %v, want ErrInvalidPayoutAttemptTransition", err)
	}
	if attempt.ProviderReference() != "transfer-123" {
		t.Error("provider reference changed after completion")
	}
}
