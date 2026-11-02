package entities_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func TestPlanPriceAtAppliesOnlyActivePromotion(t *testing.T) {
	regularPrice, err := value_objects.NewMoney(120000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)

	plan := entities.Plan{
		ID:           uuid.New(),
		Type:         "premium",
		Name:         "Premium",
		Price:        regularPrice,
		Duration:     30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{MaxProducts: 5, AIEnabled: true},
		Available:    true,
		Promotion: &entities.Promotion{
			Name:                "Launch offer",
			DiscountAmountMinor: 20000,
			StartsAt:            start,
			EndsAt:              end,
		},
	}

	tests := []struct {
		name string
		at   time.Time
		want int64
	}{
		{name: "before promotion", at: start.Add(-time.Second), want: 120000},
		{name: "during promotion", at: start, want: 100000},
		{name: "at promotion expiry", at: end, want: 120000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := plan.PriceAt(test.at)
			if err != nil {
				t.Fatalf("PriceAt() error = %v", err)
			}

			if got.AmountMinor != test.want {
				t.Errorf("PriceAt() = %d, want %d", got.AmountMinor, test.want)
			}

			if got.Currency != "NGN" {
				t.Errorf("currency = %q, want NGN", got.Currency)
			}
		})
	}

	if plan.Price.AmountMinor != 120000 {
		t.Errorf("regular plan price mutated: got %d", plan.Price.AmountMinor)
	}
}

func TestPlanRejectsPromotionDiscountAtOrAboveRegularPrice(t *testing.T) {
	price, err := value_objects.NewMoney(1000, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	plan := entities.Plan{
		ID:           uuid.New(),
		Type:         "premium",
		Name:         "Premium",
		Price:        price,
		Duration:     30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{MaxProducts: 5, AIEnabled: true},
		Promotion: &entities.Promotion{
			Name:                "Invalid offer",
			DiscountAmountMinor: 1000,
			StartsAt:            start,
			EndsAt:              start.Add(time.Hour),
		},
	}

	if err := plan.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid promotion error")
	}
}
