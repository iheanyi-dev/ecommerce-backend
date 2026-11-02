package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

type SubscriptionStatus string

const (
	SubscriptionStatusPending   SubscriptionStatus = "pending"
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusFailed    SubscriptionStatus = "failed"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

// Subscription records a store's subscription and its purchased terms.
type Subscription struct {
	ID        uuid.UUID
	StoreID   uuid.UUID
	PlanID    uuid.UUID
	Price     value_objects.Money
	Duration  time.Duration
	Status    SubscriptionStatus
	StartedAt time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewSubscription creates a pending subscription.
// Its subscription period begins only after payment verification.
func NewSubscription(
	id uuid.UUID,
	storeID uuid.UUID,
	plan Plan,
	now time.Time,
) (*Subscription, error) {
	if id == uuid.Nil ||
		storeID == uuid.Nil ||
		plan.Validate() != nil ||
		now.IsZero() {
		return nil, domain.ErrInvalidSubscription
	}

	return &Subscription{
		ID:        id,
		StoreID:   storeID,
		PlanID:    plan.ID,
		Price:     plan.Price,
		Duration:  plan.Duration,
		Status:    SubscriptionStatusPending,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	}, nil
}

// Activate starts the subscription after payment has been verified.
func (subscription *Subscription) Activate(verifiedAt time.Time) error {
	if subscription.Status != SubscriptionStatusPending {
		return domain.ErrInvalidSubscriptionStatus
	}
	if verifiedAt.IsZero() || subscription.Duration <= 0 {
		return domain.ErrInvalidSubscription
	}

	start := verifiedAt.UTC()
	subscription.StartedAt = start
	subscription.ExpiresAt = start.Add(subscription.Duration)
	subscription.Status = SubscriptionStatusActive
	subscription.UpdatedAt = start

	return nil
}

// MarkPaymentFailed records a failed initial subscription payment.
func (subscription *Subscription) MarkPaymentFailed(at time.Time) error {
	if subscription.Status != SubscriptionStatusPending {
		return domain.ErrInvalidSubscriptionStatus
	}
	if at.IsZero() {
		return domain.ErrInvalidSubscription
	}

	subscription.Status = SubscriptionStatusFailed
	subscription.UpdatedAt = at.UTC()
	return nil
}

// ExtendAfterVerifiedRenewal extends the subscription after payment verification.
// Remaining paid time is preserved when the subscription is still active.
func (subscription *Subscription) ExtendAfterVerifiedRenewal(
	verifiedAt time.Time,
) error {
	if subscription.Status != SubscriptionStatusActive &&
		subscription.Status != SubscriptionStatusExpired {
		return domain.ErrInvalidSubscriptionStatus
	}
	if verifiedAt.IsZero() || subscription.Duration <= 0 {
		return domain.ErrInvalidSubscription
	}

	verifiedAt = verifiedAt.UTC()
	start := verifiedAt

	if subscription.ExpiresAt.After(verifiedAt) {
		start = subscription.ExpiresAt
	} else {
		subscription.StartedAt = verifiedAt
	}

	subscription.ExpiresAt = start.Add(subscription.Duration)
	subscription.Status = SubscriptionStatusActive
	subscription.UpdatedAt = verifiedAt

	return nil
}

// Expire marks an active subscription expired when its expiry is reached.
func (subscription *Subscription) Expire(now time.Time) error {
	if subscription.Status != SubscriptionStatusActive {
		return domain.ErrInvalidSubscriptionStatus
	}
	if now.IsZero() {
		return domain.ErrInvalidSubscription
	}
	if now.Before(subscription.ExpiresAt) {
		return domain.ErrSubscriptionNotDue
	}

	subscription.Status = SubscriptionStatusExpired
	subscription.UpdatedAt = now.UTC()
	return nil
}

// Cancel cancels a pending or active subscription.
func (subscription *Subscription) Cancel(at time.Time) error {
	if subscription.Status != SubscriptionStatusPending &&
		subscription.Status != SubscriptionStatusActive {
		return domain.ErrInvalidSubscriptionStatus
	}
	if at.IsZero() {
		return domain.ErrInvalidSubscription
	}

	subscription.Status = SubscriptionStatusCancelled
	subscription.UpdatedAt = at.UTC()
	return nil
}
