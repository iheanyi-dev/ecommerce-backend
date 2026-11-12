package entities_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
)

func newTestPromotion(t *testing.T, productID, customerID *string) *entities.Promotion {
	t.Helper()

	percentage := int64(1000)
	discount, err := valueobjects.NewDiscount(&percentage, nil)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	promotion, err := entities.NewPromotion(
		uuid.NewString(),
		productID,
		customerID,
		discount,
		start,
		start.Add(24*time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	return promotion
}

func TestNewPromotion(t *testing.T) {
	productID := uuid.NewString()
	customerID := uuid.NewString()

	tests := []struct {
		name       string
		productID  *string
		customerID *string
	}{
		{"store-wide", nil, nil},
		{"product-specific", &productID, nil},
		{"customer store-wide", nil, &customerID},
		{"customer product-specific", &productID, &customerID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promotion := newTestPromotion(t, tt.productID, tt.customerID)
			if promotion.ID() == "" || promotion.StoreID() == "" {
				t.Fatal("promotion identifiers must be populated")
			}
		})
	}
}

func TestNewPromotionRejectsInvalidIDs(t *testing.T) {
	percentage := int64(1000)
	discount, _ := valueobjects.NewDiscount(&percentage, nil)
	start := time.Now().UTC()

	_, err := entities.NewPromotion("invalid", nil, nil, discount, start, start.Add(time.Hour))
	if !errors.Is(err, domainerrors.ErrInvalidPromotion) {
		t.Fatalf("error = %v, want ErrInvalidPromotion", err)
	}
}

func TestNewPromotionRejectsInvalidDateRange(t *testing.T) {
	percentage := int64(1000)
	discount, _ := valueobjects.NewDiscount(&percentage, nil)
	start := time.Now().UTC()

	_, err := entities.NewPromotion(uuid.NewString(), nil, nil, discount, start, start)
	if !errors.Is(err, domainerrors.ErrInvalidPromotion) {
		t.Fatalf("error = %v, want ErrInvalidPromotion", err)
	}
}

func TestPromotionStatusAt(t *testing.T) {
	promotion := newTestPromotion(t, nil, nil)
	start := promotion.StartsAt()

	tests := []struct {
		name string
		at   time.Time
		want entities.PromotionStatus
	}{
		{"scheduled", start.Add(-time.Second), entities.PromotionScheduled},
		{"active", start.Add(time.Hour), entities.PromotionActive},
		{"expired", promotion.EndsAt(), entities.PromotionExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := promotion.StatusAt(tt.at); got != tt.want {
				t.Fatalf("status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPromotionAppliesToTarget(t *testing.T) {
	productID := uuid.NewString()
	customerID := uuid.NewString()
	promotion := newTestPromotion(t, &productID, &customerID)
	at := promotion.StartsAt().Add(time.Hour)

	if !promotion.AppliesTo(promotion.StoreID(), productID, customerID, at) {
		t.Fatal("promotion should apply to matching store, product, and customer")
	}
	if promotion.AppliesTo(promotion.StoreID(), productID, uuid.NewString(), at) {
		t.Fatal("private promotion must not apply to another customer")
	}
	if promotion.AppliesTo(uuid.NewString(), productID, customerID, at) {
		t.Fatal("promotion must not apply to another store")
	}
	if promotion.AppliesTo(promotion.StoreID(), uuid.NewString(), customerID, at) {
		t.Fatal("product-specific promotion must not apply to another product")
	}
}

func TestStoreWidePromotionAppliesToAllProductsAndCustomers(t *testing.T) {
	promotion := newTestPromotion(t, nil, nil)
	at := promotion.StartsAt().Add(time.Hour)

	if !promotion.AppliesTo(promotion.StoreID(), uuid.NewString(), uuid.NewString(), at) {
		t.Fatal("store-wide promotion should apply to any product and customer in the store")
	}
}

func TestPromotionPauseAndResume(t *testing.T) {
	promotion := newTestPromotion(t, nil, nil)
	now := promotion.StartsAt().Add(time.Hour)

	if err := promotion.Pause(now); err != nil {
		t.Fatal(err)
	}
	if got := promotion.StatusAt(now); got != entities.PromotionPaused {
		t.Fatalf("status = %q, want paused", got)
	}
	if err := promotion.Resume(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got := promotion.StatusAt(now.Add(time.Minute)); got != entities.PromotionActive {
		t.Fatalf("status = %q, want active", got)
	}
}

func TestPromotionCannotBeResumedAfterExpiry(t *testing.T) {
	promotion := newTestPromotion(t, nil, nil)
	now := promotion.StartsAt().Add(time.Hour)

	if err := promotion.Pause(now); err != nil {
		t.Fatal(err)
	}
	err := promotion.Resume(promotion.EndsAt())
	if !errors.Is(err, domainerrors.ErrPromotionExpired) {
		t.Fatalf("error = %v, want ErrPromotionExpired", err)
	}
}

func TestPromotionCancel(t *testing.T) {
	promotion := newTestPromotion(t, nil, nil)
	now := promotion.StartsAt().Add(time.Hour)

	if err := promotion.Cancel(now); err != nil {
		t.Fatal(err)
	}
	if got := promotion.StatusAt(now); got != entities.PromotionCancelled {
		t.Fatalf("status = %q, want cancelled", got)
	}
	if err := promotion.Resume(now); !errors.Is(err, domainerrors.ErrPromotionCancelled) {
		t.Fatalf("resume error = %v, want ErrPromotionCancelled", err)
	}
}
