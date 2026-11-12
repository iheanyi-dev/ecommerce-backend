// Package domainerrors defines errors raised by the Product domain.
package domainerrors

import "errors"

var (
	ErrInvalidProduct     = errors.New("invalid product")
	ErrInvalidImage       = errors.New("invalid product image reference")
	ErrImageLimitExceeded = errors.New("product cannot have more than four images")
	ErrDuplicateImage     = errors.New("product image reference already exists")
	ErrImageNotFound      = errors.New("product image not found")
	ErrInvalidImageOrder  = errors.New("invalid product image order")

	ErrInvalidVideo   = errors.New("invalid product video reference")
	ErrDuplicateVideo = errors.New("product video reference already exists")
	ErrVideoNotFound  = errors.New("product video not found")
)
