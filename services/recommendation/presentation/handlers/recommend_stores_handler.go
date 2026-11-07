package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/use_cases"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/schemas"
)

// RecommendStoresHandler handles Store recommendation requests.
type RecommendStoresHandler struct {
	useCase *use_cases.RecommendStores
}

// NewRecommendStoresHandler creates a Store recommendation handler.
func NewRecommendStoresHandler(
	useCase *use_cases.RecommendStores,
) *RecommendStoresHandler {
	return &RecommendStoresHandler{useCase: useCase}
}

func (h *RecommendStoresHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		presentationerrors.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query().Get("query")
	storeIDValue := r.URL.Query().Get("store_id")

	var storeID *uuid.UUID
	if storeIDValue != "" {
		parsed, err := uuid.Parse(storeIDValue)
		if err != nil {
			presentationerrors.WriteError(w, http.StatusBadRequest, "invalid store_id")
			return
		}
		storeID = &parsed
	}

	if query == "" && storeID == nil {
		presentationerrors.WriteError(w, http.StatusBadRequest, "query or store_id is required")
		return
	}

	var queryValue *string
	if query != "" {
		queryValue = &query
	}

	result, err := h.useCase.Execute(r.Context(), dto.RecommendStoresInput{
		Query:   queryValue,
		StoreID: storeID,
	})
	if err != nil {
		presentationerrors.WriteApplicationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewStoreRecommendationResponse(
		result.Recommendations,
	))
}
