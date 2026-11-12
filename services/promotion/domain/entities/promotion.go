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

func (p *Promotion) ID() string {
	return p.id
}

func (p *Promotion) StoreID() string {
	return p.storeID
}

func (p *Promotion) ProductID() *string {
	return cloneString(p.productID)
}

func (p *Promotion) CustomerID() *string {
	return cloneString(p.customerID)
}

func (p *Promotion) Discount() valueobjects.Discount {
	return p.discount
}

func (p *Promotion) StartsAt() time.Time {
	return p.startsAt
}

func (p *Promotion) EndsAt() time.Time {
	return p.endsAt
}

func (p *Promotion) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Promotion) UpdatedAt() time.Time {
	return p.updatedAt
}

// StatusAt derives the time-dependent status while respecting explicit
// paused and cancelled states.
func (p *Promotion) StatusAt(now time.Time) PromotionStatus {
	if p.cancelled {
		return PromotionCancelled
	}
	if p.paused {
		return PromotionPaused
	}
	if now.Before(p.startsAt) {
		return PromotionScheduled
	}
	if !now.Before(p.endsAt) {
		return PromotionExpired
	}
	return PromotionActive
}

// AppliesTo reports whether this promotion targets the supplied store,
// product, and customer at the supplied time.
func (p *Promotion) AppliesTo(
	storeID string,
	productID string,
	customerID string,
	at time.Time,
) bool {
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
	if !now.Before(p.endsAt) {
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
