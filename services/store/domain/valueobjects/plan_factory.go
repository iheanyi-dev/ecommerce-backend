package valueobjects

import (
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/domain/errors"
)

// NewPlan creates the concrete plan represented by the supplied plan type.
func NewPlan(planType PlanType) (Plan, error) {
	switch planType {
	case PlanTypeBasic:
		return BasicPlan{}, nil
	case PlanTypePremium:
		return PremiumPlan{}, nil
	default:
		return nil, domainerrors.ErrInvalidPlanType
	}
}
