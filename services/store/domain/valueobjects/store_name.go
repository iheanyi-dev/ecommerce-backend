package valueobjects

import (
	"strings"
	"unicode/utf8"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

const maxStoreNameLength = 100

// StoreName represents the public name of a Store.
type StoreName struct {
	value string
}

// NewStoreName creates a valid Store name.
func NewStoreName(value string) (StoreName, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return StoreName{}, domainerrors.ErrInvalidStoreName
	}

	if utf8.RuneCountInString(value) > maxStoreNameLength {
		return StoreName{}, domainerrors.ErrInvalidStoreName
	}

	return StoreName{value: value}, nil
}

// Value returns the Store name.
func (name StoreName) Value() string {
	return name.value
}
