package entities

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

// PlanCapabilities describes the features and limits granted by a plan.
// Store remains responsible for enforcing these capabilities.
type PlanCapabilities struct {
	MaxProducts int
	AIEnabled   bool
}

// Promotion describes a discount that may apply to a plan.
type Promotion struct {
	Name                string
	DiscountAmountMinor int64
	StartsAt            time.Time
	EndsAt              time.Time
}

// IsActive reports whether the promotion is valid at the supplied time.
func (promotion Promotion) IsActive(at time.Time) bool {
	return strings.TrimSpace(promotion.Name) != "" &&
		promotion.DiscountAmountMinor > 0 &&
		!promotion.StartsAt.IsZero() &&
		promotion.EndsAt.After(promotion.StartsAt) &&
		!at.Before(promotion.StartsAt) &&
		at.Before(promotion.EndsAt)
}

// Plan describes a subscription offer managed by administrators.
type Plan struct {
	ID           uuid.UUID
	Type         string
	Name         string
	Description  string
	Price        value_objects.Money
	Duration     time.Duration
	Capabilities PlanCapabilities
	Available    bool
	Promotion    *Promotion
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate verifies that a plan has valid identity and commercial terms.
func (plan Plan) Validate() error {
	if plan.ID == uuid.Nil ||
		strings.TrimSpace(plan.Type) == "" ||
		strings.TrimSpace(plan.Name) == "" ||
		plan.Duration <= 0 ||
		plan.Capabilities.MaxProducts <= 0 {
		return domain.ErrInvalidPlan
	}

	price, err := value_objects.NewMoney(
		plan.Price.AmountMinor,
		plan.Price.Currency,
	)
	if err != nil || price != plan.Price {
		return domain.ErrInvalidPlan
	}

	if plan.Promotion != nil {
		promotion := plan.Promotion

		if strings.TrimSpace(promotion.Name) == "" ||
			promotion.DiscountAmountMinor <= 0 ||
			promotion.DiscountAmountMinor >= plan.Price.AmountMinor ||
			promotion.StartsAt.IsZero() ||
			!promotion.EndsAt.After(promotion.StartsAt) {
			return domain.ErrInvalidPlan
		}
	}

	return nil
}

// PriceAt returns the price applicable at the specified time.
func (plan Plan) PriceAt(at time.Time) (value_objects.Money, error) {
	if err := plan.Validate(); err != nil {
		return value_objects.Money{}, err
	}

	amount := plan.Price.AmountMinor

	if plan.Promotion != nil && plan.Promotion.IsActive(at) {
		amount -= plan.Promotion.DiscountAmountMinor
	}

	return value_objects.NewMoney(amount, plan.Price.Currency)
}
