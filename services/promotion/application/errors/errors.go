// Package apperrors defines errors raised by the Promotion application layer.
package apperrors

import "errors"

var (
	// ErrPromotionConflict indicates that an unexpired promotion already
	// exists for the requested store and targeting scope.
	ErrPromotionConflict = errors.New("an unexpired promotion already exists for this targeting scope")

	// ErrPromotionRepositoryRequired indicates that a required repository
	// dependency was not provided.
	ErrPromotionRepositoryRequired = errors.New("promotion repository is required")
)

var (
	// ErrInvalidPromotionQuery indicates invalid retrieval criteria.
	ErrInvalidPromotionQuery = errors.New("invalid promotion query")

	// ErrPromotionNotFound indicates no applicable promotion was found.
	ErrPromotionNotFound = errors.New("applicable promotion not found")
)
