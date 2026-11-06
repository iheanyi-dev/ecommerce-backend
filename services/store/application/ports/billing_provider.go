package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// BillingMoney represents a monetary value returned by Billing.
type BillingMoney struct {
	AmountMinor int64
	Currency    string
}

// BillingPlanPromotion represents a plan promotion.
type BillingPlanPromotion struct {
	Name                string
	DiscountAmountMinor int64
	StartsAt            *time.Time
	EndsAt              *time.Time
}

// BillingPlanCapabilities contains the capabilities included in a plan.
type BillingPlanCapabilities struct {
	MaxProducts int
	AIEnabled   bool
}

// BillingPlan contains the plan information required by Store.
type BillingPlan struct {
	ID           uuid.UUID
	Type         string
	Name         string
	Description  string
	Price        BillingMoney
	Duration     time.Duration
	Capabilities BillingPlanCapabilities
	Available    bool
	Promotion    *BillingPlanPromotion
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}

// BillingSubscription contains subscription information returned by Billing.
type BillingSubscription struct {
	ID        uuid.UUID
	StoreID   uuid.UUID
	PlanID    uuid.UUID
	Price     BillingMoney
	Duration  time.Duration
	Status    string
	StartedAt *time.Time
	ExpiresAt *time.Time
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// BillingSubscribeInput contains the information Billing needs to initiate
// a Store subscription checkout.
type BillingSubscribeInput struct {
	StoreID       uuid.UUID
	PayerID       uuid.UUID
	PlanID        uuid.UUID
	CustomerEmail string
}

// BillingSubscribeOutput contains the checkout details returned by Billing.
type BillingSubscribeOutput struct {
	SubscriptionID uuid.UUID
	PaymentID      uuid.UUID
	CheckoutURL    string
}

// BillingProvider defines the Billing capabilities required by Store.
type BillingProvider interface {
	GetPlan(ctx context.Context, planID uuid.UUID) (*BillingPlan, error)
	ListPlans(ctx context.Context) ([]BillingPlan, error)
	GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*BillingSubscription, error)
	ListStoreSubscriptions(ctx context.Context, storeID uuid.UUID) ([]BillingSubscription, error)
	Subscribe(ctx context.Context, input BillingSubscribeInput) (*BillingSubscribeOutput, error)
}
