package valueobjects

import (
	"regexp"
	"strings"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

const maxStoreSlugLength = 100

var validSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Slug represents the normalized public identifier of a Store.
type Slug struct {
	value string
}

// NewSlug creates a normalized and valid Store slug.
func NewSlug(value string) (Slug, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), "-")

	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}

	if value == "" ||
		len(value) > maxStoreSlugLength ||
		!validSlugPattern.MatchString(value) {
		return Slug{}, domainerrors.ErrInvalidStoreSlug
	}

	return Slug{value: value}, nil
}

// Value returns the normalized Store slug.
func (slug Slug) Value() string {
	return slug.value
}
