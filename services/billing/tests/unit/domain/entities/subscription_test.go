package entities_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
)

func newTestSubscription(t *testing.T, now time.Time) *entities.Subscription {
	t.Helper()

	price, err := value_objects.NewMoney(1200000, "NGN")
	if err != nil {
		t.Fatalf("create plan price: %v", err)
	}

	plan := entities.Plan{
		ID:           uuid.New(),
		Type:         "premium",
		Name:         "Premium",
		Price:        price,
		Duration:     30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{MaxProducts: 5, AIEnabled: true},
		Available:    true,
	}

	subscription, err := entities.NewSubscription(
		uuid.New(),
		uuid.New(),
		plan,
		now,
	)
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	return subscription
}

func TestNewSubscriptionStartsPendingWithoutActiveDates(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	subscription := newTestSubscription(t, now)

	if subscription.Status != entities.SubscriptionStatusPending {
		t.Fatalf("expected pending, got %q", subscription.Status)
	}
	if !subscription.StartedAt.IsZero() || !subscription.ExpiresAt.IsZero() {
		t.Fatal("pending subscription must not have an active subscription period")
	}
	if !subscription.CreatedAt.Equal(now) || !subscription.UpdatedAt.Equal(now) {
		t.Fatal("expected creation and update timestamps to use supplied time")
	}
	if subscription.Duration != 30*24*time.Hour {
		t.Fatalf("expected 30-day duration, got %v", subscription.Duration)
	}
}

func TestSubscriptionActivateStartsThirtyDayPeriod(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	verifiedAt := time.Date(2026, 1, 5, 14, 30, 0, 0, time.UTC)
	subscription := newTestSubscription(t, createdAt)

	if err := subscription.Activate(verifiedAt); err != nil {
		t.Fatalf("Activate returned error: %v", err)
	}

	if subscription.Status != entities.SubscriptionStatusActive {
		t.Fatalf("expected active, got %q", subscription.Status)
	}
	if !subscription.StartedAt.Equal(verifiedAt) {
		t.Errorf("expected start %v, got %v", verifiedAt, subscription.StartedAt)
	}
	expectedExpiry := verifiedAt.Add(30 * 24 * time.Hour)
	if !subscription.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("expected expiry %v, got %v", expectedExpiry, subscription.ExpiresAt)
	}
}

func TestSubscriptionPaymentFailureMarksPendingSubscriptionFailed(t *testing.T) {
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	subscription := newTestSubscription(t, now)

	if err := subscription.MarkPaymentFailed(now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkPaymentFailed returned error: %v", err)
	}
	if subscription.Status != entities.SubscriptionStatusFailed {
		t.Fatalf("expected failed, got %q", subscription.Status)
	}
	if !subscription.StartedAt.IsZero() || !subscription.ExpiresAt.IsZero() {
		t.Fatal("failed initial payment must not start a subscription period")
	}
}

func TestSubscriptionCannotActivateTwice(t *testing.T) {
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	subscription := newTestSubscription(t, now)

	if err := subscription.Activate(now); err != nil {
		t.Fatalf("first Activate returned error: %v", err)
	}
	if err := subscription.Activate(now.Add(time.Hour)); !errors.Is(err, domain.ErrInvalidSubscriptionStatus) {
		t.Fatalf("expected ErrInvalidSubscriptionStatus, got %v", err)
	}
}

func TestRenewalPreservesRemainingPaidTime(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	subscription := newTestSubscription(t, now)

	if err := subscription.Activate(now); err != nil {
		t.Fatal(err)
	}

	oldExpiry := subscription.ExpiresAt
	verifiedAt := now.Add(10 * 24 * time.Hour)

	if err := subscription.ExtendAfterVerifiedRenewal(verifiedAt); err != nil {
		t.Fatalf("renewal returned error: %v", err)
	}

	if subscription.Status != entities.SubscriptionStatusActive {
		t.Fatalf("expected active, got %q", subscription.Status)
	}
	expectedExpiry := oldExpiry.Add(30 * 24 * time.Hour)
	if !subscription.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("expected expiry %v, got %v", expectedExpiry, subscription.ExpiresAt)
	}
}

func TestRenewalAfterExpiryStartsFromVerificationTime(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	subscription := newTestSubscription(t, now)

	if err := subscription.Activate(now); err != nil {
		t.Fatal(err)
	}

	expiredAt := subscription.ExpiresAt
	if err := subscription.Expire(expiredAt); err != nil {
		t.Fatalf("Expire returned error: %v", err)
	}

	verifiedAt := expiredAt.Add(5 * 24 * time.Hour)
	if err := subscription.ExtendAfterVerifiedRenewal(verifiedAt); err != nil {
		t.Fatalf("renewal returned error: %v", err)
	}

	expectedExpiry := verifiedAt.Add(30 * 24 * time.Hour)
	if !subscription.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("expected expiry %v, got %v", expectedExpiry, subscription.ExpiresAt)
	}
	if subscription.Status != entities.SubscriptionStatusActive {
		t.Fatalf("expected active after renewal, got %q", subscription.Status)
	}
}

func TestSubscriptionCannotExpireBeforeExpiryTime(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	subscription := newTestSubscription(t, now)

	if err := subscription.Activate(now); err != nil {
		t.Fatal(err)
	}

	err := subscription.Expire(subscription.ExpiresAt.Add(-time.Second))
	if !errors.Is(err, domain.ErrSubscriptionNotDue) {
		t.Fatalf("expected ErrSubscriptionNotDue, got %v", err)
	}
}

func TestSubscriptionCanBeCancelledWhilePending(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	subscription := newTestSubscription(t, now)

	if err := subscription.Cancel(now.Add(time.Minute)); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if subscription.Status != entities.SubscriptionStatusCancelled {
		t.Fatalf("expected cancelled, got %q", subscription.Status)
	}
}
