package entities_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
)

func TestNewMoneyRejectsInvalidAmountsAndCurrencies(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
	}{
		{name: "zero amount", amount: 0, currency: "NGN"},
		{name: "negative amount", amount: -100, currency: "NGN"},
		{name: "unsupported currency", amount: 100, currency: "XYZ"},
		{name: "empty currency", amount: 100, currency: ""},
		{name: "whitespace currency", amount: 100, currency: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := value_objects.NewMoney(tt.amount, tt.currency)
			if !errors.Is(err, domainerrors.ErrInvalidMoney) {
				t.Errorf("error = %v, want ErrInvalidMoney", err)
			}
		})
	}
}

func TestNewMoneyNormalizesCurrency(t *testing.T) {
	money, err := value_objects.NewMoney(100, " ngn ")
	if err != nil {
		t.Fatal(err)
	}

	if money.Currency() != "NGN" {
		t.Errorf("currency = %q, want NGN", money.Currency())
	}
}

func TestRejectedPaymentTransitionsDoNotMutatePayment(t *testing.T) {
	tests := []struct {
		name   string
		action func(*entities.Payment) error
	}{
		{
			name: "succeed while pending",
			action: func(p *entities.Payment) error {
				return p.MarkSucceeded()
			},
		},
		{
			name: "fail while pending",
			action: func(p *entities.Payment) error {
				return p.MarkFailed()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := validPayment(t)
			statusBefore := payment.Status()
			updatedAtBefore := payment.UpdatedAt()

			err := tt.action(payment)
			if !errors.Is(err, domainerrors.ErrInvalidPaymentTransition) {
				t.Fatalf("error = %v, want ErrInvalidPaymentTransition", err)
			}

			if payment.Status() != statusBefore {
				t.Errorf("status changed from %q to %q after rejected transition",
					statusBefore, payment.Status())
			}

			if !payment.UpdatedAt().Equal(updatedAtBefore) {
				t.Error("UpdatedAt changed after rejected transition")
			}
		})
	}
}

func TestFailedPaymentCannotSucceed(t *testing.T) {
	payment := validPayment(t)

	if err := payment.MarkProcessing(); err != nil {
		t.Fatal(err)
	}
	if err := payment.MarkFailed(); err != nil {
		t.Fatal(err)
	}

	updatedAtBefore := payment.UpdatedAt()

	err := payment.MarkSucceeded()
	if !errors.Is(err, domainerrors.ErrInvalidPaymentTransition) {
		t.Fatalf("error = %v, want ErrInvalidPaymentTransition", err)
	}

	if payment.Status() != "failed" {
		t.Errorf("status = %q, want failed", payment.Status())
	}
	if !payment.UpdatedAt().Equal(updatedAtBefore) {
		t.Error("UpdatedAt changed after rejected transition")
	}
}
