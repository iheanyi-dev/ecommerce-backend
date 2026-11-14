// Package entities contains Promotion domain entities.
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
)

type PromotionStatus string

const (
	PromotionScheduled PromotionStatus = "scheduled"
	PromotionActive    PromotionStatus = "active"
	PromotionExpired   PromotionStatus = "expired"
	PromotionPaused    PromotionStatus = "paused"
	PromotionCancelled PromotionStatus = "cancelled"
)

// Promotion represents a seller-created discount offer.
//
// A nil ProductID targets every product in the store.
// A nil CustomerID makes the promotion available to all customers.
type Promotion struct {
	id         string
	storeID    string
	productID  *string
	customerID *string
	discount   valueobjects.Discount
	startsAt   time.Time
	endsAt     time.Time
	createdAt  time.Time
	updatedAt  time.Time
	paused     bool
	cancelled  bool
	activated  bool
	expired    bool
}

// NewPromotion creates a promotion with validated targeting and validity dates.
func NewPromotion(
	storeID string,
	productID *string,
	customerID *string,
	discount valueobjects.Discount,
	startsAt time.Time,
	endsAt time.Time,
) (*Promotion, error) {
	if _, err := uuid.Parse(storeID); err != nil {
		return nil, fmt.Errorf("%w: invalid store ID", domainerrors.ErrInvalidPromotion)
	}
	if productID != nil {
		if _, err := uuid.Parse(*productID); err != nil {
			return nil, fmt.Errorf("%w: invalid product ID", domainerrors.ErrInvalidPromotion)
		}
	}
	if customerID != nil {
		if _, err := uuid.Parse(*customerID); err != nil {
			return nil, fmt.Errorf("%w: invalid customer ID", domainerrors.ErrInvalidPromotion)
		}
	}
	if startsAt.IsZero() || endsAt.IsZero() || !endsAt.After(startsAt) {
		return nil, fmt.Errorf("%w: end time must be after start time", domainerrors.ErrInvalidPromotion)
	}

	now := time.Now().UTC()
	return &Promotion{
		id:         uuid.NewString(),
		storeID:    storeID,
		productID:  cloneString(productID),
		customerID: cloneString(customerID),
		discount:   discount,
		startsAt:   startsAt.UTC(),
		endsAt:     endsAt.UTC(),
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

func (p *Promotion) ID() string                      { return p.id }
func (p *Promotion) StoreID() string                 { return p.storeID }
func (p *Promotion) ProductID() *string              { return cloneString(p.productID) }
func (p *Promotion) CustomerID() *string             { return cloneString(p.customerID) }
func (p *Promotion) Discount() valueobjects.Discount { return p.discount }
func (p *Promotion) StartsAt() time.Time             { return p.startsAt }
func (p *Promotion) EndsAt() time.Time               { return p.endsAt }
func (p *Promotion) CreatedAt() time.Time            { return p.createdAt }
func (p *Promotion) UpdatedAt() time.Time            { return p.updatedAt }

// ReconstitutePromotion restores a promotion from persisted state.
func ReconstitutePromotion(
	id, storeID string,
	productID, customerID *string,
	discount valueobjects.Discount,
	startsAt, endsAt, createdAt, updatedAt time.Time,
	status PromotionStatus,
) (*Promotion, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("%w: invalid promotion ID", domainerrors.ErrInvalidPromotion)
	}
	if _, err := uuid.Parse(storeID); err != nil {
		return nil, fmt.Errorf("%w: invalid store ID", domainerrors.ErrInvalidPromotion)
	}
	if productID != nil {
		if _, err := uuid.Parse(*productID); err != nil {
			return nil, fmt.Errorf("%w: invalid product ID", domainerrors.ErrInvalidPromotion)
		}
	}
	if customerID != nil {
		if _, err := uuid.Parse(*customerID); err != nil {
			return nil, fmt.Errorf("%w: invalid customer ID", domainerrors.ErrInvalidPromotion)
		}
	}
	if startsAt.IsZero() || endsAt.IsZero() || !endsAt.After(startsAt) ||
		createdAt.IsZero() || updatedAt.IsZero() {
		return nil, fmt.Errorf("%w: invalid persisted promotion dates", domainerrors.ErrInvalidPromotion)
	}

	p := &Promotion{
		id:         id,
		storeID:    storeID,
		productID:  cloneString(productID),
		customerID: cloneString(customerID),
		discount:   discount,
		startsAt:   startsAt.UTC(),
		endsAt:     endsAt.UTC(),
		createdAt:  createdAt.UTC(),
		updatedAt:  updatedAt.UTC(),
	}

	switch status {
	case PromotionScheduled:
	case PromotionActive:
		p.activated = true
	case PromotionExpired:
		p.expired = true
	case PromotionPaused:
		p.paused = true
	case PromotionCancelled:
		p.cancelled = true
	default:
		return nil, fmt.Errorf("%w: invalid persisted promotion status", domainerrors.ErrInvalidPromotion)
	}

	return p, nil
}

// StatusAt derives the time-dependent status while respecting explicit states.
func (p *Promotion) StatusAt(now time.Time) PromotionStatus {
	if p.cancelled {
		return PromotionCancelled
	}
	if p.paused {
		return PromotionPaused
	}
	if p.expired || !now.Before(p.endsAt) {
		return PromotionExpired
	}
	if now.Before(p.startsAt) {
		return PromotionScheduled
	}
	return PromotionActive
}

// AppliesTo reports whether this promotion targets the supplied store,
// product, and customer at the supplied time.
func (p *Promotion) AppliesTo(storeID, productID, customerID string, at time.Time) bool {
	if p.StatusAt(at) != PromotionActive || p.storeID != storeID {
		return false
	}
	if p.productID != nil && *p.productID != productID {
		return false
	}
	if p.customerID != nil && *p.customerID != customerID {
		return false
	}
	return true
}

// Activate records that the scheduled promotion has been activated.
func (p *Promotion) Activate(now time.Time) error {
	if p.cancelled {
		return domainerrors.ErrPromotionCancelled
	}
	if p.paused {
		return domainerrors.ErrPromotionPaused
	}
	if p.expired || !now.Before(p.endsAt) {
		return domainerrors.ErrPromotionExpired
	}
	if now.Before(p.startsAt) {
		return domainerrors.ErrPromotionNotActive
	}
	if p.activated {
		return domainerrors.ErrPromotionAlreadyActive
	}

	p.activated = true
	p.updatedAt = now.UTC()
	return nil
}

// Expire explicitly marks a promotion as expired once its end time is reached.
func (p *Promotion) Expire(now time.Time) error {
	if p.cancelled {
		return domainerrors.ErrPromotionCancelled
	}
	if p.paused {
		return domainerrors.ErrPromotionPaused
	}
	if p.expired {
		return domainerrors.ErrPromotionAlreadyExpired
	}
	if now.Before(p.endsAt) {
		return domainerrors.ErrPromotionNotExpired
	}

	p.expired = true
	p.updatedAt = now.UTC()
	return nil
}

// Renew changes the validity period of an expired promotion without changing
// its identity or targeting.
func (p *Promotion) Renew(startsAt, endsAt, now time.Time) error {
	if p.cancelled {
		return domainerrors.ErrPromotionCancelled
	}
	if p.paused {
		return domainerrors.ErrPromotionPaused
	}
	if !p.expired && now.Before(p.endsAt) {
		return domainerrors.ErrPromotionNotExpired
	}
	if startsAt.IsZero() || endsAt.IsZero() || !endsAt.After(startsAt) {
		return fmt.Errorf("%w: end time must be after start time", domainerrors.ErrInvalidPromotion)
	}
	if !endsAt.After(now) {
		return fmt.Errorf("%w: renewed end time must be in the future", domainerrors.ErrInvalidPromotion)
	}

	p.startsAt = startsAt.UTC()
	p.endsAt = endsAt.UTC()
	p.expired = false
	p.activated = false
	p.updatedAt = now.UTC()
	return nil
}

// Pause pauses a scheduled or active promotion.
func (p *Promotion) Pause(now time.Time) error {
	switch p.StatusAt(now) {
	case PromotionPaused:
		return domainerrors.ErrPromotionAlreadyPaused
	case PromotionCancelled:
		return domainerrors.ErrPromotionCancelled
	case PromotionExpired:
		return domainerrors.ErrPromotionExpired
	}
	p.paused = true
	p.updatedAt = now.UTC()
	return nil
}

// Resume resumes a paused promotion, provided it has not expired or been cancelled.
func (p *Promotion) Resume(now time.Time) error {
	if p.cancelled {
		return domainerrors.ErrPromotionCancelled
	}
	if !p.paused {
		return domainerrors.ErrPromotionNotPaused
	}
	if p.expired || !now.Before(p.endsAt) {
		return domainerrors.ErrPromotionExpired
	}
	p.paused = false
	p.updatedAt = now.UTC()
	return nil
}

// Cancel permanently cancels the promotion.
func (p *Promotion) Cancel(now time.Time) error {
	if p.cancelled {
		return domainerrors.ErrPromotionCancelled
	}
	p.cancelled = true
	p.updatedAt = now.UTC()
	return nil
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
