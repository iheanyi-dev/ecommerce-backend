package valueobjects_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatus_ValidValues(t *testing.T) {
	tests := []struct {
		name   string
		status valueobjects.Status
	}{
		{
			name:   "active",
			status: valueobjects.StatusActive,
		},
		{
			name:   "inactive",
			status: valueobjects.StatusInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := valueobjects.NewStatus(tt.status)

			require.NoError(t, err)
			assert.Equal(t, tt.status, status)
		})
	}
}

func TestNewStatus_InvalidValue(t *testing.T) {
	status, err := valueobjects.NewStatus(valueobjects.Status("invalid"))

	require.Error(t, err)
	assert.Equal(t, valueobjects.Status(""), status)
}

func TestStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   valueobjects.Status
		to     valueobjects.Status
		allowed bool
	}{
		{
			name:    "active to inactive",
			from:    valueobjects.StatusActive,
			to:      valueobjects.StatusInactive,
			allowed: true,
		},
		{
			name:    "inactive to active",
			from:    valueobjects.StatusInactive,
			to:      valueobjects.StatusActive,
			allowed: true,
		},
		{
			name:    "active to active",
			from:    valueobjects.StatusActive,
			to:      valueobjects.StatusActive,
			allowed: true,
		},
		{
			name:    "inactive to inactive",
			from:    valueobjects.StatusInactive,
			to:      valueobjects.StatusInactive,
			allowed: true,
		},
		{
			name:    "active to invalid",
			from:    valueobjects.StatusActive,
			to:      valueobjects.Status("invalid"),
			allowed: false,
		},
		{
			name:    "inactive to invalid",
			from:    valueobjects.StatusInactive,
			to:      valueobjects.Status("invalid"),
			allowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.allowed, tt.from.CanTransitionTo(tt.to))
		})
	}
}
