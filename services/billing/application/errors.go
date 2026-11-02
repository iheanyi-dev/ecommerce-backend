package application

import "errors"

var (
	// ErrPlanCatalogUnavailable indicates that no plan catalogue was provided.
	ErrPlanCatalogUnavailable = errors.New("plan catalogue unavailable")

	// ErrInvalidPlanCatalogueEntry identifies a plan that cannot be displayed.
	ErrInvalidPlanCatalogueEntry = errors.New("invalid plan catalogue entry")
)
