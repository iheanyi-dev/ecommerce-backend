package valueobjects

import (
	"github.com/google/uuid"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

// OwnerID identifies the user who owns a Store.
type OwnerID struct {
	value uuid.UUID
}

// NewOwnerID creates an OwnerID from a UUID.
func NewOwnerID(value uuid.UUID) (OwnerID, error) {
	if value == uuid.Nil {
		return OwnerID{}, domainerrors.ErrInvalidOwnerID
	}

	return OwnerID{value: value}, nil
}

// Value returns the underlying UUID.
func (id OwnerID) Value() uuid.UUID {
	return id.value
}
