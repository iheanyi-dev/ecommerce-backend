package domain

import "errors"

var (
	ErrInvalidID               = errors.New("invalid ID")
	ErrInvalidDeliveryLocation = errors.New("invalid delivery location")
	ErrInvalidCurrency         = errors.New("invalid currency")
	ErrNoProducts              = errors.New("order must contain at least one product")
	ErrInvalidProduct          = errors.New("invalid order product")
	ErrInvalidPrice            = errors.New("price cannot be negative")
	ErrInvalidQuantity         = errors.New("quantity must be positive")
	ErrPriceOverflow           = errors.New("order price overflow")
	ErrInvalidState            = errors.New("invalid order state")
	ErrInvalidTransition       = errors.New("invalid order state transition")
	ErrUnauthorizedActor       = errors.New("actor is not authorized")
	ErrInvalidShippingAmount   = errors.New("invalid shipping amount")
)
