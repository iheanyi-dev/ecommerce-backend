package valueobjects_test

import (
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlug_NormalizesValue(t *testing.T) {
	slug, err := valueobjects.NewSlug("  My  Awesome---Store  ")

	require.NoError(t, err)
	assert.Equal(t, "my-awesome-store", slug.Value())
}

func TestNewSlug_AcceptsLettersNumbersAndHyphens(t *testing.T) {
	slug, err := valueobjects.NewSlug("store-123")

	require.NoError(t, err)
	assert.Equal(t, "store-123", slug.Value())
}

func TestNewSlug_RejectsEmptyValue(t *testing.T) {
	slug, err := valueobjects.NewSlug("   ")

	require.Error(t, err)
	assert.Equal(t, "", slug.Value())
}

func TestNewSlug_RejectsInvalidCharacters(t *testing.T) {
	tests := []string{
		"my_store",
		"my/store",
		"my@store",
		"my.store",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			slug, err := valueobjects.NewSlug(input)

			require.Error(t, err)
			assert.Equal(t, "", slug.Value())
		})
	}
}

func TestNewSlug_RejectsLeadingOrTrailingHyphen(t *testing.T) {
	tests := []string{
		"-my-store",
		"my-store-",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			slug, err := valueobjects.NewSlug(input)

			require.Error(t, err)
			assert.Equal(t, "", slug.Value())
		})
	}
}

func TestNewSlug_RejectsValueLongerThanMaximum(t *testing.T) {
	slug, err := valueobjects.NewSlug(strings.Repeat("a", 101))

	require.Error(t, err)
	assert.Equal(t, "", slug.Value())
}

func TestNewSlug_AcceptsMaximumLength(t *testing.T) {
	slug, err := valueobjects.NewSlug(strings.Repeat("a", 100))

	require.NoError(t, err)
	assert.Len(t, slug.Value(), 100)
}
