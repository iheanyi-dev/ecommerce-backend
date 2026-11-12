package value_objects

import "github.com/iheanyi-dev/ecommerce-backend/services/notification/domain"

type EmailFormat string

const (
	EmailFormatPlainText EmailFormat = "text/plain"
	EmailFormatHTML      EmailFormat = "text/html"
)

func NewEmailFormat(value string) (EmailFormat, error) {
	format := EmailFormat(value)

	switch format {
	case EmailFormatPlainText, EmailFormatHTML:
		return format, nil
	default:
		return "", domain.ErrInvalidEmailFormat
	}
}

func (f EmailFormat) Value() string {
	return string(f)
}
