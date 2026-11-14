package apperrors

import "errors"

var (
	ErrPromotionOutboxRepositoryRequired = errors.New(
		"promotion outbox repository is required",
	)

	ErrPromotionUnitOfWorkRequired = errors.New(
		"promotion unit of work is required",
	)
)
