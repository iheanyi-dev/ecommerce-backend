package user

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrInvalidFullName = errors.New("invalid full name")

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
		return FullName{}, ErrInvalidFullName
	}

	return FullName{value: value}, nil
}

// String returns the full name.
func (name FullName) String() string {
	return name.value
}
