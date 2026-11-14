// Package domainerrors defines errors raised by the Promotion domain.
package domainerrors

import "errors"

var (
	ErrInvalidPromotion        = errors.New("invalid promotion")
	ErrPromotionAlreadyPaused  = errors.New("promotion is already paused")
	ErrPromotionNotPaused      = errors.New("promotion is not paused")
	ErrPromotionExpired        = errors.New("promotion has expired")
	ErrPromotionCancelled      = errors.New("promotion is cancelled")
	ErrPromotionAlreadyActive  = errors.New("promotion is already activated")
	ErrPromotionNotActive      = errors.New("promotion is not active")
	ErrPromotionAlreadyExpired = errors.New("promotion is already expired")
	ErrPromotionNotExpired     = errors.New("promotion has not expired")
	ErrPromotionPaused         = errors.New("promotion is paused")
)
