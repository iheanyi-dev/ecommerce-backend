package entities_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func TestPlanValidateRejectsInvalidTerms(t *testing.T) {
	tests := []struct {
		name string
		plan entities.Plan
	}{
		{
			name: "empty type",
			plan: entities.Plan{
				Price:    value_objects.Money{AmountMinor: 100, Currency: "NGN"},
				Duration: 30 * 24 * time.Hour,
			},
		},
		{
			name: "negative price",
			plan: entities.Plan{
				Type:     "premium",
				Price:    value_objects.Money{AmountMinor: -1, Currency: "NGN"},
				Duration: 30 * 24 * time.Hour,
			},
		},
		{
			name: "invalid currency",
			plan: entities.Plan{
				Type:     "premium",
				Price:    value_objects.Money{AmountMinor: 100, Currency: "N1G"},
				Duration: 30 * 24 * time.Hour,
			},
		},
		{
			name: "zero duration",
			plan: entities.Plan{
				Type:  "premium",
				Price: value_objects.Money{AmountMinor: 100, Currency: "NGN"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.plan.Validate(); err == nil {
				t.Fatal("expected invalid plan to be rejected")
			}
		})
	}
}

func TestPlanValidateAcceptsValidTerms(t *testing.T) {
	plan := entities.Plan{
		ID:   uuid.New(),
		Type: "premium",
		Name: "Premium",
		Price: value_objects.Money{
			AmountMinor: 10000,
			Currency:    "NGN",
		},
		Duration:     30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{MaxProducts: 5, AIEnabled: true},
	}

	if err := plan.Validate(); err != nil {
		t.Fatalf("expected valid plan, got %v", err)
	}
}
