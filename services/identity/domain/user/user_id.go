package user

import (
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidUserID = errors.New("invalid user ID")

// UserID uniquely identifies a User within the Identity domain.
//
// The UUID is wrapped in a value object so that the domain does not pass
// unvalidated identifier strings between its components.
type UserID struct {
	value uuid.UUID
}

// NewUserID generates a new unique UserID.
func NewUserID() UserID {
	return UserID{
		value: uuid.New(),
	}
}

// UserIDFromString reconstructs a UserID from its persisted representation.
func UserIDFromString(value string) (UserID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return UserID{}, ErrInvalidUserID
	}

	return UserID{value: id}, nil
}

// String returns the canonical string representation of the UserID.
func (id UserID) String() string {
	return id.value.String()
}

func (id UserID) Value() uuid.UUID {
	return id.value
}
