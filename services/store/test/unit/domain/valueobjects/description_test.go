package valueobjects_test

import (
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDescription(t *testing.T) {
	description, err := valueobjects.NewDescription("  A store description.  ")

	require.NoError(t, err)
	assert.Equal(t, "A store description.", description.Value())
}

func TestNewDescription_AllowsEmptyDescription(t *testing.T) {
	description, err := valueobjects.NewDescription("   ")

	require.NoError(t, err)
	assert.Equal(t, "", description.Value())
}

func TestNewDescription_RejectsDescriptionLongerThanMaximum(t *testing.T) {
	description, err := valueobjects.NewDescription(strings.Repeat("a", 501))

	require.Error(t, err)
	assert.Equal(t, "", description.Value())
}

func TestNewDescription_AcceptsMaximumLength(t *testing.T) {
	description, err := valueobjects.NewDescription(strings.Repeat("a", 500))

	require.NoError(t, err)
	assert.Len(t, description.Value(), 500)
}
