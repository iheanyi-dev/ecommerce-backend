package valueobjects

import (
	"strings"
	"unicode/utf8"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

const maxStoreDescriptionLength = 500

// Description represents the optional public description of a Store.
type Description struct {
	value string
}

// NewDescription creates a valid Store description.
func NewDescription(value string) (Description, error) {
	value = strings.TrimSpace(value)

	if utf8.RuneCountInString(value) > maxStoreDescriptionLength {
		return Description{}, domainerrors.ErrInvalidStoreDescription
	}

	return Description{value: value}, nil
}

// Value returns the Store description.
func (description Description) Value() string {
	return description.value
}
