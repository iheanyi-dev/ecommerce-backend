package user

import (
	"strings"
	"unicode/utf8"

	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
)

const maxFullNameLength = 150

// FullName represents the display name associated with a User.
//
// The value object guarantees that the domain never receives an empty or
// excessively long full name.
type FullName struct {
	value string
}

// NewFullName creates a validated FullName.
func NewFullName(value string) (FullName, error) {
	value = strings.TrimSpace(value)

	if value == "" || utf8.RuneCountInString(value) > maxFullNameLength {
		return FullName{}, domain_errors.ErrInvalidFullName
	}

	return FullName{value: value}, nil
}

// String returns the full name.
func (name FullName) String() string {
	return name.value
}
