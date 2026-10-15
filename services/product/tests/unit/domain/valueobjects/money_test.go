package valueobjects_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
		wantErr  bool
	}{
		{name: "valid NGN amount", amount: 1500000, currency: "NGN"},
		{name: "zero amount", amount: 0, currency: "NGN"},
		{name: "negative amount", amount: -1, currency: "NGN", wantErr: true},
		{name: "missing currency", amount: 100, currency: "", wantErr: true},
		{name: "unsupported currency", amount: 100, currency: "XXX", wantErr: true},
		{name: "lowercase currency", amount: 100, currency: "ngn", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := valueobjects.NewMoney(tt.amount, tt.currency)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewMoney() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
