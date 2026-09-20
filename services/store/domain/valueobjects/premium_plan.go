package valueobjects

const premiumProductLimit = 5

// PremiumPlan defines the entitlements of the Premium Store plan.
type PremiumPlan struct{}

func (PremiumPlan) Type() PlanType {
	return PlanTypePremium
}

func (PremiumPlan) ProductLimit() int {
	return premiumProductLimit
}

func (PremiumPlan) CanAccommodateProducts(productCount int) bool {
	return productCount >= 0 && productCount <= premiumProductLimit
}

func (PremiumPlan) AIFeaturesAllowed() bool {
	return true
}
