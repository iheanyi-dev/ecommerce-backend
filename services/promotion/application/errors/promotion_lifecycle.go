package apperrors

import "errors"

var (
	// ErrPromotionNotDeletable indicates that a promotion is neither expired nor cancelled.
	ErrPromotionNotDeletable = errors.New("only expired or cancelled promotions can be deleted")
)
