package schemas

import "github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"

// ListStoresResponse contains publicly discoverable Stores and pagination
// metadata.
type ListStoresResponse struct {
	Stores     []StoreResponse `json:"stores"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	Total      int             `json:"total"`
	TotalPages int             `json:"total_pages"`
}

// NewListStoresResponse maps the application result to the HTTP response.
func NewListStoresResponse(result *dto.ListStoresOutput) ListStoresResponse {
	stores := make([]StoreResponse, 0, len(result.Stores))

	for _, store := range result.Stores {
		stores = append(stores, NewStoreResponse(store))
	}

	return ListStoresResponse{
		Stores:     stores,
		Page:       result.Page,
		PageSize:   result.PageSize,
		Total:      result.Total,
		TotalPages: result.TotalPages,
	}
}
