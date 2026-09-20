package errors

import "errors"

var (
	ErrInvalidPlanType         = errors.New("invalid plan type")
	ErrInvalidStoreID          = errors.New("invalid store ID")
	ErrInvalidOwnerID          = errors.New("invalid owner ID")
	ErrInvalidStatus           = errors.New("invalid store status")
	ErrInvalidStatusTransition = errors.New("invalid store status transition")
	ErrInvalidStoreName        = errors.New("invalid store name")
	ErrInvalidStoreDescription = errors.New("invalid store description")
	ErrInvalidStoreSlug        = errors.New("invalid store slug")
	ErrInvalidStoreTimestamps  = errors.New("invalid store timestamps")
)
