package user

import (
	"errors"
	"strings"
)

var ErrInvalidPasswordHash = errors.New("invalid password hash")

// PasswordHash represents a password after it has been securely hashed.
//
// The domain deliberately does not perform password hashing. Hashing is an
// infrastructure concern and will later be exposed through an application
// boundary. This prevents the User aggregate from depending on a specific
// hashing algorithm or library.
type PasswordHash struct {
	value string
}

// NewPasswordHash creates a PasswordHash from an already-hashed password.
func NewPasswordHash(value string) (PasswordHash, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return PasswordHash{}, ErrInvalidPasswordHash
	}

	return PasswordHash{value: value}, nil
}

// String returns the stored password hash.
func (passwordHash PasswordHash) String() string {
	return passwordHash.value
}
