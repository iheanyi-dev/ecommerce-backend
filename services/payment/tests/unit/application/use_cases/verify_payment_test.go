package use_cases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
)

type verificationFixture struct {
	*fixture
	verify    *use_cases.VerifyPaymentUseCase
	paymentID string
}

func newVerificationFixture(t *testing.T) *verificationFixture {
	t.Helper()

	f := newFixture(t)
	created, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("initiate payment: %v", err)
	}

	verify, err := use_cases.NewVerifyPaymentUseCase(
		f.payments, f.attempts, f.outbox, f.provider, f.uow,
	)
	if err != nil {
		t.Fatal(err)
	}

	f.provider.reconcileResult = ports.ReconciliationResult{
		ProviderReference: "provider-ref-001",
		Amount:            f.input.Amount.Amount(),
		Currency:          f.input.Amount.Currency(),
		Status:            enums.PaymentAttemptStatusSucceeded,
	}

	return &verificationFixture{fixture: f, verify: verify, paymentID: created.PaymentID}
}

func assertPaymentStatusEvent(t *testing.T, f *verificationFixture, want enums.PaymentStatus) {
	t.Helper()

	if len(f.outbox.events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(f.outbox.events))
	}

	event := f.outbox.events[0]
	var payload struct {
		EventID               string              `json:"event_id"`
		BusinessReferenceType string              `json:"business_reference_type"`
		BusinessReference     string              `json:"business_reference"`
		Status                enums.PaymentStatus `json:"status"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}
	if payload.EventID != event.ID {
		t.Errorf("payload event ID = %q, want %q", payload.EventID, event.ID)
	}
	if payload.BusinessReferenceType != "order" {
		t.Errorf("business reference type = %q, want order", payload.BusinessReferenceType)
	}
	if payload.BusinessReference != testOrderID {
		t.Errorf("business reference = %q, want %q", payload.BusinessReference, testOrderID)
	}
	if payload.Status != want {
		t.Errorf("event status = %q, want %q", payload.Status, want)
	}
}

func TestVerifyPaymentMarksPaymentAndAttemptSucceeded(t *testing.T) {
	f := newVerificationFixture(t)

	got, err := f.verify.Execute(context.Background(), f.paymentID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PaymentStatusSucceeded {
		t.Errorf("payment status = %q, want succeeded", got.Status)
	}
	if got.Attempt == nil || got.Attempt.Status != enums.PaymentAttemptStatusSucceeded {
		t.Fatalf("attempt details = %+v, want succeeded", got.Attempt)
	}
	assertPaymentStatusEvent(t, f, enums.PaymentStatusSucceeded)
	if f.provider.reconcileCalls != 1 {
		t.Errorf("reconcile calls = %d, want 1", f.provider.reconcileCalls)
	}
}

func TestVerifyPaymentMarksPaymentAndAttemptFailed(t *testing.T) {
	f := newVerificationFixture(t)
	f.provider.reconcileResult.Status = enums.PaymentAttemptStatusFailed
	f.provider.reconcileResult.FailureReason = "card declined"

	got, err := f.verify.Execute(context.Background(), f.paymentID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PaymentStatusFailed {
		t.Errorf("payment status = %q, want failed", got.Status)
	}
	if got.Attempt == nil || got.Attempt.Status != enums.PaymentAttemptStatusFailed {
		t.Fatalf("attempt details = %+v, want failed", got.Attempt)
	}
	if got.Attempt.FailureReason != "card declined" {
		t.Errorf("failure reason = %q", got.Attempt.FailureReason)
	}
	assertPaymentStatusEvent(t, f, enums.PaymentStatusFailed)
}

func TestVerifyPaymentKeepsPaymentProcessingForInconclusiveResult(t *testing.T) {
	f := newVerificationFixture(t)
	f.provider.reconcileResult.Status = enums.PaymentAttemptStatusProcessing

	got, err := f.verify.Execute(context.Background(), f.paymentID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != enums.PaymentStatusProcessing {
		t.Errorf("payment status = %q, want processing", got.Status)
	}
	if got.Attempt.Status != enums.PaymentAttemptStatusProcessing {
		t.Errorf("attempt status = %q, want processing", got.Attempt.Status)
	}
	if len(f.outbox.events) != 0 {
		t.Errorf("outbox events = %d, want 0 for nonterminal payment", len(f.outbox.events))
	}
}

func TestVerifyPaymentDoesNotFailPaymentWhenProviderErrors(t *testing.T) {
	f := newVerificationFixture(t)
	f.provider.reconcileErr = errors.New("provider unavailable")

	_, err := f.verify.Execute(context.Background(), f.paymentID)
	if err == nil {
		t.Fatal("expected provider error")
	}
	payment, getErr := f.payments.GetByID(context.Background(), f.paymentID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if payment.Status() != enums.PaymentStatusProcessing {
		t.Errorf("payment status = %q, want processing", payment.Status())
	}
}

func TestVerifyPaymentRejectsMismatchedProviderReference(t *testing.T) {
	f := newVerificationFixture(t)
	f.provider.reconcileResult.ProviderReference = "another-transaction"

	_, err := f.verify.Execute(context.Background(), f.paymentID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPaymentRejectsMismatchedAmount(t *testing.T) {
	f := newVerificationFixture(t)
	f.provider.reconcileResult.Amount++

	_, err := f.verify.Execute(context.Background(), f.paymentID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPaymentRejectsMismatchedCurrency(t *testing.T) {
	f := newVerificationFixture(t)
	f.provider.reconcileResult.Currency = "USD"

	_, err := f.verify.Execute(context.Background(), f.paymentID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPaymentRejectsFailureWithoutReason(t *testing.T) {
	f := newVerificationFixture(t)
	f.provider.reconcileResult.Status = enums.PaymentAttemptStatusFailed
	f.provider.reconcileResult.FailureReason = ""

	_, err := f.verify.Execute(context.Background(), f.paymentID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPaymentRejectsInvalidPaymentIDBeforeProviderCall(t *testing.T) {
	f := newVerificationFixture(t)

	_, err := f.verify.Execute(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected invalid UUID error")
	}
	if f.provider.reconcileCalls != 0 {
		t.Errorf("provider calls = %d, want 0", f.provider.reconcileCalls)
	}
}

func TestVerifyPaymentTerminalReplayDoesNotCallProviderAgain(t *testing.T) {
	f := newVerificationFixture(t)

	if _, err := f.verify.Execute(context.Background(), f.paymentID); err != nil {
		t.Fatal(err)
	}
	assertPaymentStatusEvent(t, f, enums.PaymentStatusSucceeded)
	calls := f.provider.reconcileCalls

	got, err := f.verify.Execute(context.Background(), f.paymentID)
	if err != nil {
		t.Fatalf("replayed Execute() error = %v", err)
	}
	if got.Status != enums.PaymentStatusSucceeded {
		t.Errorf("status = %q, want succeeded", got.Status)
	}
	if f.provider.reconcileCalls != calls {
		t.Errorf("reconcile calls = %d, want unchanged at %d", f.provider.reconcileCalls, calls)
	}
	assertPaymentStatusEvent(t, f, enums.PaymentStatusSucceeded)
}

func TestVerifyPaymentRejectsTerminalAttemptWithNonterminalPayment(t *testing.T) {
	f := newVerificationFixture(t)
	ctx := context.Background()

	payment, err := f.payments.GetByID(ctx, f.paymentID)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := f.attempts.GetByID(ctx, payment.AttemptID())
	if err != nil {
		t.Fatal(err)
	}

	if attempt.Status() == enums.PaymentAttemptStatusPending {
		if err := attempt.MarkProcessing(); err != nil {
			t.Fatal(err)
		}
	}
	if err := attempt.MarkSucceeded(); err != nil {
		t.Fatal(err)
	}
	if err := f.attempts.Save(ctx, attempt); err != nil {
		t.Fatal(err)
	}

	_, err = f.verify.Execute(ctx, f.paymentID)
	if !errors.Is(err, application.ErrInvalidProviderResponse) {
		t.Fatalf("error = %v, want ErrInvalidProviderResponse", err)
	}
}

func TestVerifyPaymentReturnsOutboxPersistenceError(t *testing.T) {
	f := newVerificationFixture(t)
	f.outbox.err = errors.New("outbox unavailable")

	_, err := f.verify.Execute(context.Background(), f.paymentID)
	if err == nil {
		t.Fatal("expected outbox persistence error")
	}
	if !errors.Is(err, f.outbox.err) {
		t.Errorf("error = %v, want outbox persistence error", err)
	}
}
