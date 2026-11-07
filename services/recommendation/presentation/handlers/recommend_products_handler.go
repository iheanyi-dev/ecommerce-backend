package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/use_cases"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/presentation/schemas"
)

// RecommendProductsHandler handles Product recommendation requests.
type RecommendProductsHandler struct {
	useCase *use_cases.RecommendProducts
}

// NewRecommendProductsHandler creates a Product recommendation handler.
func NewRecommendProductsHandler(
	useCase *use_cases.RecommendProducts,
) *RecommendProductsHandler {
	return &RecommendProductsHandler{useCase: useCase}
}

func (h *RecommendProductsHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		presentationerrors.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query().Get("query")
	productIDValue := r.URL.Query().Get("product_id")

	var productID *uuid.UUID
	if productIDValue != "" {
		parsed, err := uuid.Parse(productIDValue)
		if err != nil {
			presentationerrors.WriteError(w, http.StatusBadRequest, "invalid product_id")
			return
		}
		productID = &parsed
	}

	if query == "" && productID == nil {
		presentationerrors.WriteError(w, http.StatusBadRequest, "query or product_id is required")
		return
	}

	var queryValue *string
	if query != "" {
		queryValue = &query
	}

	result, err := h.useCase.Execute(r.Context(), dto.RecommendProductsInput{
		Query:     queryValue,
		ProductID: productID,
	})
	if err != nil {
		presentationerrors.WriteApplicationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, schemas.NewProductRecommendationResponse(
		result.Recommendations,
	))
}
