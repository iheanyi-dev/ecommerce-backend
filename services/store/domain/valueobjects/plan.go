package valueobjects

// PlanType identifies a Store subscription plan.
type PlanType string

const (
	PlanTypeBasic   PlanType = "basic"
	PlanTypePremium PlanType = "premium"
)

// Plan defines the behavior and entitlements of a Store plan.
type Plan interface {
	Type() PlanType
	ProductLimit() int
	CanAccommodateProducts(productCount int) bool
	AIFeaturesAllowed() bool
}
