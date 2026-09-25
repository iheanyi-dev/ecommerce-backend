package schemas

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// StoreResponse is the public HTTP representation of a Store.
//
// Domain value objects are deliberately flattened here because the
// presentation layer owns the transport representation.
type StoreResponse struct {
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

// NewStoreResponse maps the application Store DTO to its HTTP
// representation.
func NewStoreResponse(store dto.StoreOutput) StoreResponse {
	return StoreResponse{
		ID:             store.ID.Value().String(),
		OwnerID:        store.OwnerID.Value().String(),
		Name:           store.Name.Value(),
		Slug:           store.Slug.Value(),
		Description:    store.Description.Value(),
		ImageReference: store.ImageReference,
		Status:         string(store.Status),
		Plan:           string(store.Plan.Type()),
		CreatedAt:      store.CreatedAt,
		UpdatedAt:      store.UpdatedAt,
	}
}
