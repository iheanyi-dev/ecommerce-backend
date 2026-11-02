package valueobjects_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStoreName(t *testing.T) {
	name, err := valueobjects.NewStoreName("  My Store  ")

	require.NoError(t, err)
	assert.Equal(t, "My Store", name.Value())
}

func TestNewStoreName_RejectsEmptyName(t *testing.T) {
	name, err := valueobjects.NewStoreName("   ")

	require.Error(t, err)
	assert.Equal(t, "", name.Value())
}

func TestNewStoreName_RejectsNameLongerThanMaximum(t *testing.T) {
	name, err := valueobjects.NewStoreName(string(make([]byte, 101)))

	require.Error(t, err)
	assert.Equal(t, "", name.Value())
}

func TestNewStoreName_AcceptsMaximumLength(t *testing.T) {
	name, err := valueobjects.NewStoreName(string(make([]byte, 100)))

	require.NoError(t, err)
	assert.Len(t, name.Value(), 100)
}
