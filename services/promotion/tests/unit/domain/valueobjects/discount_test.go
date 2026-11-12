package valueobjects_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
)

func TestNewDiscount(t *testing.T) {
	percentage := int64(1000)
	fixed, _ := valueobjects.NewMoney(15000, "NGN")

	tests := []struct {
		name       string
		percentage *int64
		fixed      *valueobjects.Money
		wantErr    bool
	}{
		{"percentage only", &percentage, nil, false},
		{"fixed only", nil, &fixed, false},
		{"both types", &percentage, &fixed, false},
		{"neither type", nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := valueobjects.NewDiscount(tt.percentage, tt.fixed)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewDiscountRejectsInvalidPercentage(t *testing.T) {
	for _, percentage := range []int64{0, -1, 10001} {
		_, err := valueobjects.NewDiscount(&percentage, nil)
		if !errors.Is(err, valueobjects.ErrInvalidDiscount) {
			t.Fatalf("percentage %d: error = %v", percentage, err)
		}
	}
}

func TestNewDiscountRejectsZeroFixedAmount(t *testing.T) {
	fixed, _ := valueobjects.NewMoney(0, "NGN")

	_, err := valueobjects.NewDiscount(nil, &fixed)
	if !errors.Is(err, valueobjects.ErrInvalidDiscount) {
		t.Fatalf("error = %v, want ErrInvalidDiscount", err)
	}
}

func TestDiscountApplyPercentage(t *testing.T) {
	percentage := int64(1500)
	discount, err := valueobjects.NewDiscount(&percentage, nil)
	if err != nil {
		t.Fatal(err)
	}

	price, _ := valueobjects.NewMoney(100000, "NGN")
	got, err := discount.Apply(price)
	if err != nil {
		t.Fatal(err)
	}

	if got.Amount() != 85000 {
		t.Fatalf("amount = %d, want 85000", got.Amount())
	}
}

func TestDiscountApplyFixedTakesPrecedence(t *testing.T) {
	percentage := int64(5000)
	fixed, _ := valueobjects.NewMoney(12000, "NGN")
	discount, err := valueobjects.NewDiscount(&percentage, &fixed)
	if err != nil {
		t.Fatal(err)
	}

	price, _ := valueobjects.NewMoney(100000, "NGN")
	got, err := discount.Apply(price)
	if err != nil {
		t.Fatal(err)
	}

	if got.Amount() != 88000 {
		t.Fatalf("amount = %d, want 88000; fixed discount should take precedence", got.Amount())
	}
}

func TestDiscountApplyDoesNotProduceNegativePrice(t *testing.T) {
	fixed, _ := valueobjects.NewMoney(150000, "NGN")
	discount, err := valueobjects.NewDiscount(nil, &fixed)
	if err != nil {
		t.Fatal(err)
	}

	price, _ := valueobjects.NewMoney(100000, "NGN")
	got, err := discount.Apply(price)
	if err != nil {
		t.Fatal(err)
	}

	if got.Amount() != 0 {
		t.Fatalf("amount = %d, want 0", got.Amount())
	}
}

func TestDiscountApplyRejectsCurrencyMismatch(t *testing.T) {
	fixed, _ := valueobjects.NewMoney(1000, "USD")
	discount, err := valueobjects.NewDiscount(nil, &fixed)
	if err != nil {
		t.Fatal(err)
	}

	price, _ := valueobjects.NewMoney(100000, "NGN")
	_, err = discount.Apply(price)
	if !errors.Is(err, valueobjects.ErrDiscountCurrencyMismatch) {
		t.Fatalf("error = %v, want currency mismatch", err)
	}
}

func TestDiscountApplyPreservesCurrency(t *testing.T) {
	percentage := int64(1000)
	discount, _ := valueobjects.NewDiscount(&percentage, nil)
	price, _ := valueobjects.NewMoney(100000, "NGN")

	got, err := discount.Apply(price)
	if err != nil {
		t.Fatal(err)
	}
	if got.Currency() != "NGN" {
		t.Fatalf("currency = %q, want NGN", got.Currency())
	}
}
