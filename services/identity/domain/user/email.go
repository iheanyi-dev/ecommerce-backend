package user

import (
	"net/mail"
	"strings"

	domain_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/errors"
)

// Email represents a normalized and validated email address.
//
// Email is immutable from the perspective of the domain. Once constructed,
// consumers can rely on it satisfying the Email value object's invariants.
type Email struct {
	value string
}

// NewEmail validates and normalizes an email address.
func NewEmail(value string) (Email, error) {
	value = strings.ToLower(strings.TrimSpace(value))

	if value == "" {
		return Email{}, domain_errors.ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return Email{}, domain_errors.ErrInvalidEmail
	}

	return Email{value: value}, nil
}

// String returns the normalized email address.
func (email Email) String() string {
	return email.value
}
