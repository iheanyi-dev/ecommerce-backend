package valueobjects_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStoreID(t *testing.T) {
	id := uuid.New()

	storeID, err := valueobjects.NewStoreID(id)

	require.NoError(t, err)
	assert.Equal(t, id, storeID.Value())
}

func TestNewStoreID_RejectsNilUUID(t *testing.T) {
	storeID, err := valueobjects.NewStoreID(uuid.Nil)

	require.Error(t, err)
	assert.Equal(t, uuid.Nil, storeID.Value())
}
