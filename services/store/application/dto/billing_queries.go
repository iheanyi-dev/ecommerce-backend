package dto

import (
	"time"

	"github.com/google/uuid"
)

// BillingMoneyOutput represents a monetary amount returned by Billing.
type BillingMoneyOutput struct {
	AmountMinor int64
	Currency    string
}

// BillingPromotionOutput represents a plan promotion.
type BillingPromotionOutput struct {
	Name                string
	DiscountAmountMinor int64
	StartsAt            *time.Time
	EndsAt              *time.Time
}

// BillingPlanOutput represents a plan available in Billing.
type BillingPlanOutput struct {
	ID          uuid.UUID
	Type        string
	Name        string
	Description string
	Price       BillingMoneyOutput
	Duration    time.Duration
	MaxProducts int
	AIEnabled   bool
	Available   bool
	Promotion   *BillingPromotionOutput
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

// BillingSubscriptionOutput represents a Store subscription.
type BillingSubscriptionOutput struct {
	ID        uuid.UUID
	StoreID   uuid.UUID
	PlanID    uuid.UUID
	Price     BillingMoneyOutput
	Duration  time.Duration
	Status    string
	StartedAt *time.Time
	ExpiresAt *time.Time
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// GetBillingPlanInput identifies a Billing plan.
type GetBillingPlanInput struct {
	PlanID uuid.UUID
}

// ListStoreSubscriptionsInput identifies the Store whose subscriptions
// should be retrieved.
type ListStoreSubscriptionsInput struct {
	StoreID uuid.UUID
}

// GetStoreSubscriptionInput identifies a Store and one of its subscriptions.
type GetStoreSubscriptionInput struct {
	StoreID        uuid.UUID
	SubscriptionID uuid.UUID
}
