package valueobjects_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlan_Basic(t *testing.T) {
	plan, err := valueobjects.NewPlan(valueobjects.PlanTypeBasic)

	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, valueobjects.PlanTypeBasic, plan.Type())
	assert.Equal(t, 2, plan.ProductLimit())
	assert.False(t, plan.AIFeaturesAllowed())
}

func TestNewPlan_Premium(t *testing.T) {
	plan, err := valueobjects.NewPlan(valueobjects.PlanTypePremium)

	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, valueobjects.PlanTypePremium, plan.Type())
	assert.Equal(t, 5, plan.ProductLimit())
	assert.True(t, plan.AIFeaturesAllowed())
}

func TestNewPlan_InvalidType(t *testing.T) {
	plan, err := valueobjects.NewPlan(valueobjects.PlanType("invalid"))

	require.Error(t, err)
	assert.Nil(t, plan)
}

func TestPlan_CanAccommodateProducts(t *testing.T) {
	tests := []struct {
		name         string
		planType     valueobjects.PlanType
		productCount int
		expected     bool
	}{
		{
			name:         "basic at limit",
			planType:     valueobjects.PlanTypeBasic,
			productCount: 2,
			expected:     true,
		},
		{
			name:         "basic exceeds limit",
			planType:     valueobjects.PlanTypeBasic,
			productCount: 3,
			expected:     false,
		},
		{
			name:         "premium at limit",
			planType:     valueobjects.PlanTypePremium,
			productCount: 5,
			expected:     true,
		},
		{
			name:         "premium exceeds limit",
			planType:     valueobjects.PlanTypePremium,
			productCount: 6,
			expected:     false,
		},
		{
			name:         "negative count",
			planType:     valueobjects.PlanTypeBasic,
			productCount: -1,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := valueobjects.NewPlan(tt.planType)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, plan.CanAccommodateProducts(tt.productCount))
		})
	}
}
