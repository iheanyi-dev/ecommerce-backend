package versioning_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/versioning"
	"github.com/stretchr/testify/assert"
)

func TestPrefix(t *testing.T) {
	assert.Equal(t, "/api/v1", versioning.Prefix(versioning.V1))
}
