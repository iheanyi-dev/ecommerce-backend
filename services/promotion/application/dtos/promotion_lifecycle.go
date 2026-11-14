package dtos

import "time"

// CancelPromotionInput identifies a promotion to cancel.
type CancelPromotionInput struct {
	PromotionID string
}

// ActivatePromotionInput identifies a promotion to activate.
type ActivatePromotionInput struct {
	PromotionID string
}

// ExpirePromotionInput identifies a promotion to expire.
type ExpirePromotionInput struct {
	PromotionID string
}

// RenewPromotionInput contains the replacement promotion period.
type RenewPromotionInput struct {
	PromotionID string
	StartsAt    time.Time
	EndsAt      time.Time
}

// DeletePromotionInput identifies a promotion to delete.
type DeletePromotionInput struct {
	PromotionID string
}
