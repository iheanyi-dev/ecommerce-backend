package valueobjects

const basicProductLimit = 2

// BasicPlan defines the entitlements of the Basic Store plan.
type BasicPlan struct{}

func (BasicPlan) Type() PlanType {
	return PlanTypeBasic
}

func (BasicPlan) ProductLimit() int {
	return basicProductLimit
}

func (BasicPlan) CanAccommodateProducts(productCount int) bool {
	return productCount >= 0 && productCount <= basicProductLimit
}

func (BasicPlan) AIFeaturesAllowed() bool {
	return false
}
