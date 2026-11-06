package errors

import "errors"

var (
	ErrUnauthenticated               = errors.New("unauthenticated")
	ErrInvalidServiceCredentials     = errors.New("invalid service credentials")
	ErrStoreAlreadyExists            = errors.New("store already exists")
	ErrStoreSlugAlreadyExists        = errors.New("store slug already exists")
	ErrStoreNotFound                 = errors.New("store not found")
	ErrInvalidPagination             = errors.New("invalid pagination")
	ErrStorePlanProductLimitExceeded = errors.New("store plan product limit exceeded")
	ErrStoreProductCapacityExceeded  = errors.New("store product capacity exceeded")
	ErrStoreProductCountUnderflow    = errors.New("store product count cannot be negative")
	ErrInvalidSubscriptionRequest    = errors.New("invalid subscription request")
	ErrBillingPlanNotAvailable       = errors.New("billing plan is not available")
	ErrUnsupportedStorePlan          = errors.New("unsupported store subscription plan")
	ErrIdentityUserUnavailable       = errors.New("identity user information unavailable")
	ErrBillingCheckoutUnavailable    = errors.New("billing checkout unavailable")
	ErrBillingSubscriptionNotFound   = errors.New("billing subscription not found")
)
