package domain

import "errors"

var (
	ErrInvalidMoney              = errors.New("invalid money")
	ErrInvalidPlan               = errors.New("invalid plan")
	ErrUnsupportedPlan           = errors.New("unsupported plan")
	ErrInvalidSubscription       = errors.New("invalid subscription")
	ErrInvalidSubscriptionStatus = errors.New("invalid subscription status")
	ErrSubscriptionNotDue        = errors.New("subscription has not expired")
)
