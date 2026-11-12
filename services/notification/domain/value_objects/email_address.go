package value_objects

import (
	"net/mail"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain"
)

type EmailAddress struct {
	value string
}

func NewEmailAddress(value string) (EmailAddress, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return EmailAddress{}, domain.ErrInvalidEmailAddress
	}

	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return EmailAddress{}, domain.ErrInvalidEmailAddress
	}

	return EmailAddress{value: value}, nil
}

func (e EmailAddress) Value() string {
	return e.value
}
