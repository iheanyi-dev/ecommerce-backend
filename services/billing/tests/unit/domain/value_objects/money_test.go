package value_objects_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func TestNewMoney(t *testing.T) {
	money, err := value_objects.NewMoney(12500, "ngn")
	if err != nil {
		t.Fatalf("NewMoney returned error: %v", err)
	}
	if money.AmountMinor != 12500 {
		t.Errorf("expected amount 12500, got %d", money.AmountMinor)
	}
	if money.Currency != "NGN" {
		t.Errorf("expected currency NGN, got %q", money.Currency)
	}
}

func TestNewMoneyRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
	}{
		{name: "negative amount", amount: -1, currency: "NGN"},
		{name: "empty currency", amount: 100, currency: ""},
		{name: "short currency", amount: 100, currency: "NG"},
		{name: "non-letter currency", amount: 100, currency: "N1G"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := value_objects.NewMoney(test.amount, test.currency)
			if !errors.Is(err, domain.ErrInvalidMoney) {
				t.Fatalf("expected ErrInvalidMoney, got %v", err)
			}
		})
	}
}
