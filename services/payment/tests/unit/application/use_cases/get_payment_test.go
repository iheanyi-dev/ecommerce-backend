package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/use_cases"
)

func TestGetPaymentByIDReturnsPaymentAndAttempt(t *testing.T) {
	f := newFixture(t)
	created, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("seed payment: %v", err)
	}

	uc, err := use_cases.NewGetPaymentByIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	got, err := uc.Execute(context.Background(), created.PaymentID)
	if err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if got.ID != created.PaymentID {
		t.Errorf("payment ID = %q, want %q", got.ID, created.PaymentID)
	}
	if got.Attempt == nil {
		t.Fatal("expected current attempt details")
	}
	if got.Attempt.ID != created.AttemptID {
		t.Errorf("attempt ID = %q, want %q", got.Attempt.ID, created.AttemptID)
	}
	if got.Attempt.CheckoutURL == "" {
		t.Error("expected checkout URL")
	}
	if got.Amount != f.input.Amount.Amount() || got.Currency != f.input.Amount.Currency() {
		t.Errorf("unexpected amount/currency: %d %s", got.Amount, got.Currency)
	}
}

func TestGetPaymentByIDRejectsInvalidUUID(t *testing.T) {
	f := newFixture(t)
	uc, err := use_cases.NewGetPaymentByIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.Execute(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected UUID validation error")
	}
}

func TestGetPaymentByIDReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	uc, err := use_cases.NewGetPaymentByIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.Execute(context.Background(), testOrderID)
	if !errors.Is(err, application.ErrPaymentNotFound) {
		t.Fatalf("error = %v, want ErrPaymentNotFound", err)
	}
}

func TestGetPaymentByBusinessReferenceReturnsPaymentAndAttempt(t *testing.T) {
	f := newFixture(t)
	created, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("seed payment: %v", err)
	}

	uc, err := use_cases.NewGetPaymentByBusinessReferenceUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	got, err := uc.Execute(context.Background(), f.input.BusinessRefType, f.input.BusinessRefID)
	if err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if got.ID != created.PaymentID {
		t.Errorf("payment ID = %q, want %q", got.ID, created.PaymentID)
	}
	if got.Attempt == nil || got.Attempt.ID != created.AttemptID {
		t.Fatal("expected matching current attempt details")
	}
	if got.BusinessReferenceType != f.input.BusinessRefType ||
		got.BusinessReferenceID != f.input.BusinessRefID {
		t.Errorf("unexpected business reference: %s:%s",
			got.BusinessReferenceType, got.BusinessReferenceID)
	}
}

func TestGetPaymentByBusinessReferenceRejectsInvalidUUID(t *testing.T) {
	f := newFixture(t)
	uc, err := use_cases.NewGetPaymentByBusinessReferenceUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.Execute(context.Background(), "order", "bad-id")
	if err == nil {
		t.Fatal("expected UUID validation error")
	}
}

func TestGetPaymentByBusinessReferenceReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	uc, err := use_cases.NewGetPaymentByBusinessReferenceUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.Execute(context.Background(), "order", testOrderID)
	if !errors.Is(err, application.ErrPaymentNotFound) {
		t.Fatalf("error = %v, want ErrPaymentNotFound", err)
	}
}

func TestGetPaymentReturnsAttemptNotFoundWhenCurrentAttemptIsMissing(t *testing.T) {
	f := newFixture(t)
	created, err := f.uc.Execute(context.Background(), f.input)
	if err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	delete(f.attempts.items, created.AttemptID)

	uc, err := use_cases.NewGetPaymentByIDUseCase(f.payments, f.attempts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.Execute(context.Background(), created.PaymentID)
	if !errors.Is(err, application.ErrPaymentAttemptNotFound) {
		t.Fatalf("error = %v, want ErrPaymentAttemptNotFound", err)
	}
}
