// Package valueobjects defines immutable value objects used by the Product domain.
package valueobjects

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidMoney indicates that a monetary amount or currency is invalid.
var ErrInvalidMoney = errors.New("invalid money")

// Money represents a monetary amount in the smallest currency unit.
//
// Amount uses integer minor units to avoid floating-point precision errors.
// For example, 1500000 NGN minor units represents ₦15,000.00.
//
// Money is immutable: its fields are private and accessors return values.
type Money struct {
	amount   int64
	currency string
}

// supportedCurrencies lists ISO 4217 currency codes supported by this domain.
// Add currencies deliberately as the marketplace expands.
var supportedCurrencies = map[string]struct{}{
	"NGN": {},
	"USD": {},
	"EUR": {},
	"GBP": {},
	"JPY": {},
	"KES": {},
	"GHS": {},
	"ZAR": {},
	"CAD": {},
	"AUD": {},
	"INR": {},
	"CNY": {},
}

// NewMoney creates a monetary value in integer minor units.
//
// Negative amounts and unsupported or incorrectly formatted currency codes
// are rejected. Currency codes must be uppercase ISO 4217 codes.
func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, fmt.Errorf("%w: amount cannot be negative", ErrInvalidMoney)
	}

	if len(currency) != 3 || currency != strings.ToUpper(currency) {
		return Money{}, fmt.Errorf("%w: currency must be a three-letter uppercase code", ErrInvalidMoney)
	}

	if _, ok := supportedCurrencies[currency]; !ok {
		return Money{}, fmt.Errorf("%w: unsupported currency %q", ErrInvalidMoney, currency)
	}

	return Money{
		amount:   amount,
		currency: currency,
	}, nil
}

// Amount returns the amount in the currency's smallest unit.
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the ISO 4217 currency code.
func (m Money) Currency() string {
	return m.currency
}

// Equal reports whether two monetary values have the same amount and currency.
func (m Money) Equal(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}
