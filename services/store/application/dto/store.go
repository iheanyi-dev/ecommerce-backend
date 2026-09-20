package dto

import (
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
)

// StoreOutput represents Store data returned by the application layer.
//
// The application layer preserves the domain value objects rather than
// flattening them into presentation-specific primitive values.
type StoreOutput struct {
	ID             valueobjects.StoreID
	OwnerID        valueobjects.OwnerID
	Name           valueobjects.StoreName
	Slug           valueobjects.Slug
	Description    valueobjects.Description
	ImageReference *string
	Status         valueobjects.Status
	Plan           valueobjects.Plan
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
