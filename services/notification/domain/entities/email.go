package entities

import (
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/notification/domain/value_objects"
)

type Email struct {
	recipient value_objects.EmailAddress
	subject   string
	body      string
	format    value_objects.EmailFormat
}

func NewEmail(
	recipient value_objects.EmailAddress,
	subject string,
	body string,
	format value_objects.EmailFormat,
) (Email, error) {
	if recipient.Value() == "" {
		return Email{}, domain.ErrInvalidEmailAddress
	}

	if strings.TrimSpace(subject) == "" {
		return Email{}, domain.ErrEmptyEmailSubject
	}

	if strings.TrimSpace(body) == "" {
		return Email{}, domain.ErrEmptyEmailBody
	}

	if format != value_objects.EmailFormatPlainText &&
		format != value_objects.EmailFormatHTML {
		return Email{}, domain.ErrInvalidEmailFormat
	}

	return Email{
		recipient: recipient,
		subject:   subject,
		body:      body,
		format:    format,
	}, nil
}

func (e Email) Recipient() value_objects.EmailAddress {
	return e.recipient
}

func (e Email) Subject() string {
	return e.subject
}

func (e Email) Body() string {
	return e.body
}

func (e Email) Format() value_objects.EmailFormat {
	return e.format
}
