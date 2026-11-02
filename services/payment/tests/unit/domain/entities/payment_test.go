package entities_test

import (
	"errors"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

func validPayment(t *testing.T) *entities.Payment {
	t.Helper()

	amount, err := value_objects.NewMoney(2500000, "NGN")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	businessReference, err := value_objects.NewBusinessReference(
		"order",
		"order-123",
	)
	if err != nil {
		t.Fatalf("NewBusinessReference() error = %v", err)
	}

	payment, err := entities.NewPayment(
		"customer-123",
		enums.PaymentPurposeOrder,
		businessReference,
		amount,
	)
	if err != nil {
		t.Fatalf("NewPayment() error = %v", err)
	}

	return payment
}

func TestNewPayment(t *testing.T) {
	payment := validPayment(t)

	if payment.ID() == "" {
		t.Fatal("ID must not be empty")
	}
	if payment.Reference() == "" {
		t.Fatal("Reference must not be empty")
	}
	if payment.ID() == payment.Reference() {
		t.Fatal("ID and Reference must be distinct")
	}
	if payment.PayerID() != "customer-123" {
		t.Errorf("PayerID() = %q, want customer-123", payment.PayerID())
	}
	if payment.Purpose() != enums.PaymentPurposeOrder {
		t.Errorf("Purpose() = %q, want order", payment.Purpose())
	}
	if payment.Amount().Amount() != 2500000 {
		t.Errorf("Amount() = %d, want 2500000", payment.Amount().Amount())
	}
	if payment.Status() != enums.PaymentStatusPending {
		t.Errorf("Status() = %q, want pending", payment.Status())
	}
	if payment.CreatedAt().IsZero() || payment.UpdatedAt().IsZero() {
		t.Fatal("creation and update timestamps must be set")
	}
	if payment.CreatedAt().Location() != time.UTC {
		t.Error("CreatedAt must use UTC")
	}
	if payment.UpdatedAt().Before(payment.CreatedAt()) {
		t.Error("UpdatedAt must not precede CreatedAt")
	}
}

func TestNewPaymentRejectsInvalidInput(t *testing.T) {
	amount, err := value_objects.NewMoney(1000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	businessReference, err := value_objects.NewBusinessReference("order", "order-123")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name              string
		payerID           string
		purpose           enums.PaymentPurpose
		businessReference value_objects.BusinessReference
	}{
		{
			name:              "empty payer",
			payerID:           " ",
			purpose:           enums.PaymentPurposeOrder,
			businessReference: businessReference,
		},
		{
			name:              "invalid purpose",
			payerID:           "customer-123",
			purpose:           enums.PaymentPurpose("unknown"),
			businessReference: businessReference,
		},
		{
			name:              "empty business reference",
			payerID:           "customer-123",
			purpose:           enums.PaymentPurposeOrder,
			businessReference: value_objects.BusinessReference{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.NewPayment(
				tt.payerID,
				tt.purpose,
				tt.businessReference,
				amount,
			)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestPaymentLifecycle(t *testing.T) {
	payment := validPayment(t)

	if err := payment.MarkProcessing(); err != nil {
		t.Fatalf("MarkProcessing() error = %v", err)
	}
	if payment.Status() != enums.PaymentStatusProcessing {
		t.Fatalf("status = %q, want processing", payment.Status())
	}

	if err := payment.MarkSucceeded(); err != nil {
		t.Fatalf("MarkSucceeded() error = %v", err)
	}
	if payment.Status() != enums.PaymentStatusSucceeded {
		t.Fatalf("status = %q, want succeeded", payment.Status())
	}

	if err := payment.MarkFailed(); !errors.Is(err, domainerrors.ErrInvalidPaymentTransition) {
		t.Errorf("MarkFailed() after success error = %v, want invalid transition", err)
	}
}

func TestPaymentFailureLifecycle(t *testing.T) {
	payment := validPayment(t)

	if err := payment.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := payment.MarkFailed(); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if payment.Status() != enums.PaymentStatusFailed {
		t.Errorf("Status() = %q, want failed", payment.Status())
	}
}

func TestPaymentRejectsInvalidTransitions(t *testing.T) {
	tests := []struct {
		name   string
		action func(*entities.Payment) error
	}{
		{"succeed while pending", func(p *entities.Payment) error {
			return p.MarkSucceeded()
		}},
		{"fail while pending", func(p *entities.Payment) error {
			return p.MarkFailed()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := validPayment(t)
			if err := tt.action(payment); !errors.Is(err, domainerrors.ErrInvalidPaymentTransition) {
				t.Errorf("error = %v, want invalid transition", err)
			}
		})
	}
}

func TestPaymentUpdatedAtChangesOnTransition(t *testing.T) {
	payment := validPayment(t)
	before := payment.UpdatedAt()

	time.Sleep(time.Millisecond)

	if err := payment.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if !payment.UpdatedAt().After(before) {
		t.Error("UpdatedAt must advance after a successful transition")
	}
}
