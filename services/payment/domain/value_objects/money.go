package value_objects

import (
	"strings"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/errors"
)

type Money struct {
	amount   int64
	currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if amount <= 0 || !isSupportedCurrency(currency) {
		return Money{}, domainerrors.ErrInvalidMoney
	}

	return Money{
		amount:   amount,
		currency: currency,
	}, nil
}

func (m Money) Amount() int64 {
	return m.amount
}

func (m Money) Currency() string {
	return m.currency
}

// IsZero reports whether the value fails the domain's positive-money invariant.
func (m Money) IsZero() bool {
	return m.amount <= 0 || !isSupportedCurrency(m.currency)
}

func isSupportedCurrency(currency string) bool {
	switch currency {
	case "NGN", "USD", "EUR", "GBP", "JPY",
		"KES", "GHS", "ZAR", "CAD", "AUD",
		"INR", "CNY":
		return true
	default:
		return false
	}
}
