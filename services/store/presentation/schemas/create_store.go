package schemas

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// CreateStoreResponse represents the public HTTP response returned after
// successfully creating a Store.
//
// Domain value objects are converted to primitive values at the presentation
// boundary so the HTTP API does not expose domain implementation details.
type CreateStoreResponse struct {
	ID             string    `json:"id"`
	OwnerID        string    `json:"owner_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	ImageReference *string   `json:"image_reference,omitempty"`
	Status         string    `json:"status"`
	Plan           string    `json:"plan"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewCreateStoreResponse converts the application result into the HTTP
// response schema.
func NewCreateStoreResponse(
	result dto.CreateStoreOutput,
) CreateStoreResponse {
	return CreateStoreResponse{
		ID:             result.ID.Value().String(),
		OwnerID:        result.OwnerID.Value().String(),
		Name:           result.Name.Value(),
		Slug:           result.Slug.Value(),
		Description:    result.Description.Value(),
		ImageReference: result.ImageReference,
		Status:         string(result.Status),
		Plan:           string(result.Plan.Type()),
		CreatedAt:      result.CreatedAt,
		UpdatedAt:      result.UpdatedAt,
	}
}
