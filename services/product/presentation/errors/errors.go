// Package presentationerrors maps Product application errors to HTTP responses.
package presentationerrors

import (
	"encoding/json"
	"errors"
	"net/http"

	appusecases "github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/domain/errors"
)

// Write writes a stable JSON error response for a Product HTTP endpoint.
func Write(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, appusecases.ErrProductNotFound):
		status = http.StatusNotFound
		message = "product not found"

	case errors.Is(err, appusecases.ErrProductOwnerStoreNotFound):
		status = http.StatusNotFound
		message = "store not found"

	case errors.Is(err, appusecases.ErrInvalidProductOwner):
		status = http.StatusUnauthorized
		message = "authentication required"

	case errors.Is(err, appusecases.ErrInvalidPagination),
		errors.Is(err, appusecases.ErrInvalidProductFilters),
		errors.Is(err, appusecases.ErrInvalidMarketplaceProductSearch),
		errors.Is(err, appusecases.ErrInvalidStoreProductSearch),
		errors.Is(err, appusecases.ErrInvalidProductUpdate),
		errors.Is(err, appusecases.ErrInvalidProductDeletion),
		errors.Is(err, appusecases.ErrInvalidProductImage),
		errors.Is(err, appusecases.ErrInvalidProductIDs),
		errors.Is(err, domainerrors.ErrInvalidProduct),
		errors.Is(err, domainerrors.ErrInvalidImage),
		errors.Is(err, domainerrors.ErrImageLimitExceeded),
		errors.Is(err, domainerrors.ErrDuplicateImage),
		errors.Is(err, domainerrors.ErrImageNotFound),
		errors.Is(err, domainerrors.ErrInvalidImageOrder):
		status = http.StatusBadRequest
		message = "invalid product request"

	case errors.Is(err, appusecases.ErrProductStoreInactive):
		status = http.StatusConflict
		message = "store is inactive"

	case errors.Is(err, appusecases.ErrProductCreationOutcomeUnknown):
		status = http.StatusInternalServerError
		message = "product creation outcome is unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
