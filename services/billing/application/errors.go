package application

import "errors"

var (
	ErrPlanRepositoryUnavailable = errors.New("plan repository unavailable")
	ErrPlanNotFound              = errors.New("plan not found")
	ErrPlanAlreadyExists         = errors.New("plan already exists")
	ErrPlanInUse                 = errors.New("plan is referenced by subscriptions")

	ErrSubscriptionRepositoryUnavailable = errors.New("subscription repository unavailable")
	ErrSubscriptionNotFound              = errors.New("subscription not found")
	ErrSubscriptionConflict              = errors.New("subscription conflicts with an existing subscription")
	ErrActiveSubscriptionExists          = errors.New("store already has an active subscription")
	ErrSubscriptionAlreadyPending        = errors.New("store already has a pending subscription; cancel it before choosing another plan")
	ErrPaymentUnavailable                = errors.New("payment service unavailable")
	ErrPaymentInitiationFailed           = errors.New("subscription payment initiation failed")
	ErrInvalidPaymentResponse            = errors.New("invalid payment initiation response")
	ErrPlanUnavailable                   = errors.New("subscription plan is unavailable")

	ErrStoreNotifierUnavailable = errors.New("store subscription notifier unavailable")
	ErrStoreNotificationFailed  = errors.New("store subscription notification failed")
)
