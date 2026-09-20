package valueobjects

import (
	"github.com/google/uuid"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

// StoreID uniquely identifies a Store.
type StoreID struct {
	value uuid.UUID
}

// NewStoreID creates a StoreID from a UUID.
func NewStoreID(value uuid.UUID) (StoreID, error) {
	if value == uuid.Nil {
		return StoreID{}, domainerrors.ErrInvalidStoreID
	}

	return StoreID{value: value}, nil
}

// Value returns the underlying UUID.
func (id StoreID) Value() uuid.UUID {
	return id.value
}
