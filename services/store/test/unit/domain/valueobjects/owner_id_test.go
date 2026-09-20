package valueobjects_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOwnerID(t *testing.T) {
	id := uuid.New()

	ownerID, err := valueobjects.NewOwnerID(id)

	require.NoError(t, err)
	assert.Equal(t, id, ownerID.Value())
}

func TestNewOwnerID_RejectsNilUUID(t *testing.T) {
	ownerID, err := valueobjects.NewOwnerID(uuid.Nil)

	require.Error(t, err)
	assert.Equal(t, uuid.Nil, ownerID.Value())
}
