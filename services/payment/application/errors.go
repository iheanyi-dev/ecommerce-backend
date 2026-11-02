package application

import "errors"

var (
	ErrPaymentNotFound               = errors.New("payment not found")
	ErrPaymentAttemptNotFound        = errors.New("payment attempt not found")
	ErrPaymentAlreadyExists          = errors.New("payment already exists")
	ErrIdempotencyConflict           = errors.New("payment reference reused with different details")
	ErrAttemptRequiresReconciliation = errors.New("payment attempt requires reconciliation")
	ErrInvalidProviderResponse       = errors.New("invalid payment provider response")
	ErrInvalidPagination             = errors.New("invalid pagination parameters")
)

var (
	ErrPayoutNotFound            = errors.New("payout not found")
	ErrPayoutAttemptNotFound     = errors.New("payout attempt not found")
	ErrPayoutAlreadyExists       = errors.New("payout already exists")
	ErrPayoutIdempotencyConflict = errors.New("payout reference reused with different details")
	ErrPayoutRetryNotAllowed     = errors.New("payout retry requires an explicitly failed attempt")
)
