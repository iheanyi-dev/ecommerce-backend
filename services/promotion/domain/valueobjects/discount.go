// Package valueobjects defines immutable value objects used by the Promotion domain.
package valueobjects

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidDiscount          = errors.New("invalid discount")
	ErrDiscountCurrencyMismatch = errors.New("discount currency does not match product currency")
)

// Discount represents an optional percentage and/or fixed-amount discount.
//
// PercentageBasisPoints uses 1/100th of one percent.
// For example, 10% is represented as 1000 basis points.
//
// When both values are configured, FixedAmount takes precedence.
type Discount struct {
	percentageBasisPoints *int64
	fixedAmount           *Money
}

// NewDiscount creates a discount with at least one configured discount type.
func NewDiscount(percentageBasisPoints *int64, fixedAmount *Money) (Discount, error) {
	if percentageBasisPoints == nil && fixedAmount == nil {
		return Discount{}, fmt.Errorf("%w: at least one discount type is required", ErrInvalidDiscount)
	}

	var percentageCopy *int64
	if percentageBasisPoints != nil {
		if *percentageBasisPoints < 1 || *percentageBasisPoints > 10000 {
			return Discount{}, fmt.Errorf("%w: percentage must be between 1 and 10000 basis points", ErrInvalidDiscount)
		}

		value := *percentageBasisPoints
		percentageCopy = &value
	}

	var fixedCopy *Money
	if fixedAmount != nil {
		if fixedAmount.Amount() <= 0 {
			return Discount{}, fmt.Errorf("%w: fixed amount must be greater than zero", ErrInvalidDiscount)
		}

		value := *fixedAmount
		fixedCopy = &value
	}

	return Discount{
		percentageBasisPoints: percentageCopy,
		fixedAmount:           fixedCopy,
	}, nil
}

// PercentageBasisPoints returns the configured percentage, if present.
func (d Discount) PercentageBasisPoints() (int64, bool) {
	if d.percentageBasisPoints == nil {
		return 0, false
	}
	return *d.percentageBasisPoints, true
}

// FixedAmount returns the configured fixed discount, if present.
func (d Discount) FixedAmount() (Money, bool) {
	if d.fixedAmount == nil {
		return Money{}, false
	}
	return *d.fixedAmount, true
}

// Apply calculates the final price after applying the discount.
//
// A fixed discount takes precedence over a percentage discount.
// The final price is never negative.
func (d Discount) Apply(price Money) (Money, error) {
	if d.fixedAmount != nil {
		if d.fixedAmount.Currency() != price.Currency() {
			return Money{}, ErrDiscountCurrencyMismatch
		}

		remaining := price.Amount() - d.fixedAmount.Amount()
		if remaining < 0 {
			remaining = 0
		}

		return NewMoney(remaining, price.Currency())
	}

	if d.percentageBasisPoints == nil {
		return Money{}, ErrInvalidDiscount
	}

	// Split the calculation to avoid overflowing price.Amount()*basisPoints.
	amount := price.Amount()
	basisPoints := *d.percentageBasisPoints

	discountAmount := (amount/10000)*basisPoints +
		((amount%10000)*basisPoints)/10000

	return NewMoney(amount-discountAmount, price.Currency())
}
