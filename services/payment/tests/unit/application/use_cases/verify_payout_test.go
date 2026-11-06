package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
)

type payoutVerificationFixture struct {
	*payoutFixture
	verify   *use_cases.VerifyPayoutUseCase
	payoutID string
}

func newPayoutVerificationFixture(t *testing.T) *payoutVerificationFixture {
	t.Helper()

	f := newPayoutFixture(t)
	created, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("initiate payout: %v", err)
	}

	verify, err := use_cases.NewVerifyPayoutUseCase(
		f.payouts, f.attempts, f.provider, f.outbox, f.uow,
	)
	if err != nil {
		t.Fatal(err)
	}

	f.provider.reconcileResult = ports.PayoutReconciliationResult{
		ProviderReference: "transfer-ref-001",
		Amount:            f.input.Amount.Amount(),
		Currency:          f.input.Amount.Currency(),
		Status:            enums.PayoutAttemptStatusSucceeded,
	}
	return &payoutVerificationFixture{
		payoutFixture: f,
		verify:        verify,
		payoutID:      created.PayoutID,
	}
}

func TestVerifyPayoutMarksPayoutAndAttemptSucceeded(t *testing.T) {
	f := newPayoutVerificationFixture(t)

	got, err := f.verify.Execute(context.Background(), f.payoutID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PayoutStatusSucceeded {
		t.Errorf("payout status = %q, want succeeded", got.Status)
	}
	if got.AttemptStatus != enums.PayoutAttemptStatusSucceeded {
		t.Errorf("attempt status = %q, want succeeded", got.AttemptStatus)
	}
	if f.provider.reconcileCalls != 1 {
		t.Errorf("reconcile calls = %d, want 1", f.provider.reconcileCalls)
	}
}

func TestVerifyPayoutMarksPayoutAndAttemptFailed(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileResult.Status = enums.PayoutAttemptStatusFailed
	f.provider.reconcileResult.FailureReason = "bank rejected transfer"

	got, err := f.verify.Execute(context.Background(), f.payoutID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PayoutStatusFailed {
		t.Errorf("payout status = %q, want failed", got.Status)
	}
	if got.AttemptStatus != enums.PayoutAttemptStatusFailed {
		t.Errorf("attempt status = %q, want failed", got.AttemptStatus)
	}
	attempt, err := f.attempts.GetByID(context.Background(), got.AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.FailureReason() != "bank rejected transfer" {
		t.Errorf("failure reason = %q", attempt.FailureReason())
	}
}

func TestVerifyPayoutKeepsTransferProcessingForInconclusiveResult(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileResult.Status = enums.PayoutAttemptStatusProcessing

	got, err := f.verify.Execute(context.Background(), f.payoutID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PayoutStatusProcessing {
		t.Errorf("payout status = %q, want processing", got.Status)
	}
	if got.AttemptStatus != enums.PayoutAttemptStatusProcessing {
		t.Errorf("attempt status = %q, want processing", got.AttemptStatus)
	}
}

func TestVerifyPayoutDoesNotChangeStateWhenProviderErrors(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileErr = errors.New("provider unavailable")

	_, err := f.verify.Execute(context.Background(), f.payoutID)
	if err == nil {
		t.Fatal("expected provider error")
	}
	payout, getErr := f.payouts.GetByID(context.Background(), f.payoutID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if payout.Status() != enums.PayoutStatusProcessing {
		t.Errorf("payout status = %q, want processing", payout.Status())
	}
}

func TestVerifyPayoutRejectsMismatchedProviderReference(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileResult.ProviderReference = "different-transfer"

	_, err := f.verify.Execute(context.Background(), f.payoutID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPayoutRejectsMismatchedAmount(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileResult.Amount++

	_, err := f.verify.Execute(context.Background(), f.payoutID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPayoutRejectsMismatchedCurrency(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileResult.Currency = "USD"

	_, err := f.verify.Execute(context.Background(), f.payoutID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPayoutRejectsFailureWithoutReason(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileResult.Status = enums.PayoutAttemptStatusFailed
	f.provider.reconcileResult.FailureReason = ""

	_, err := f.verify.Execute(context.Background(), f.payoutID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPayoutTerminalReplayDoesNotCallProviderAgain(t *testing.T) {
	f := newPayoutVerificationFixture(t)

	if _, err := f.verify.Execute(context.Background(), f.payoutID); err != nil {
		t.Fatal(err)
	}
	calls := f.provider.reconcileCalls

	got, err := f.verify.Execute(context.Background(), f.payoutID)
	if err != nil {
		t.Fatalf("replayed Execute() error = %v", err)
	}
	if got.Status != enums.PayoutStatusSucceeded {
		t.Errorf("status = %q, want succeeded", got.Status)
	}
	if f.provider.reconcileCalls != calls {
		t.Errorf("reconcile calls = %d, want unchanged at %d", f.provider.reconcileCalls, calls)
	}
}

func TestVerifyPayoutRejectsInvalidIDBeforeProviderCall(t *testing.T) {
	f := newPayoutVerificationFixture(t)

	_, err := f.verify.Execute(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected invalid UUID error")
	}
	if f.provider.reconcileCalls != 0 {
		t.Errorf("provider calls = %d, want 0", f.provider.reconcileCalls)
	}
}

func TestVerifyPayoutEmitsEventOnlyOnceWhenSucceeded(t *testing.T) {
	f := newPayoutVerificationFixture(t)

	if _, err := f.verify.Execute(context.Background(), f.payoutID); err != nil {
		t.Fatalf("first verification: %v", err)
	}
	if len(f.outbox.events) != 1 {
		t.Fatalf("outbox events after success = %d, want 1", len(f.outbox.events))
	}

	if _, err := f.verify.Execute(context.Background(), f.payoutID); err != nil {
		t.Fatalf("repeated verification: %v", err)
	}
	if len(f.outbox.events) != 1 {
		t.Fatalf("outbox events after repeated verification = %d, want 1", len(f.outbox.events))
	}
}

func TestVerifyPayoutEmitsEventWhenFailed(t *testing.T) {
	f := newPayoutVerificationFixture(t)
	f.provider.reconcileResult.Status = enums.PayoutAttemptStatusFailed
	f.provider.reconcileResult.FailureReason = "bank rejected transfer"

	got, err := f.verify.Execute(context.Background(), f.payoutID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PayoutStatusFailed {
		t.Fatalf("payout status = %q, want failed", got.Status)
	}
	if len(f.outbox.events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(f.outbox.events))
	}
}
