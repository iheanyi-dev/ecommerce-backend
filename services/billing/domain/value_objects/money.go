package value_objects

import (
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
)

// Money represents an amount in minor currency units.
type Money struct {
	AmountMinor int64
	Currency    string
}

// NewMoney constructs a validated monetary value.
func NewMoney(amountMinor int64, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if amountMinor < 0 || len(currency) != 3 {
		return Money{}, domain.ErrInvalidMoney
	}

	for _, character := range currency {
		if character < 'A' || character > 'Z' {
			return Money{}, domain.ErrInvalidMoney
		}
	}

	return Money{
		AmountMinor: amountMinor,
		Currency:    currency,
	}, nil
}
