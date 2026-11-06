package use_cases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
)

const subscriptionUpdatedEventType = "billing.subscription.updated"

type subscriptionUpdatedEvent struct {
	EventID        string                      `json:"event_id"`
	SubscriptionID string                      `json:"subscription_id"`
	StoreID        string                      `json:"store_id"`
	PlanType       string                      `json:"plan_type"`
	Status         entities.SubscriptionStatus `json:"status"`
	StartedAt      time.Time                   `json:"started_at"`
	ExpiresAt      time.Time                   `json:"expires_at"`
}

func recordSubscriptionUpdate(
	ctx context.Context,
	outbox ports.SubscriptionOutboxRepository,
	plans ports.PlanRepository,
	subscription entities.Subscription,
) error {
	if plans == nil {
		return fmt.Errorf("plan repository is unavailable")
	}
	plan, err := plans.GetByID(ctx, subscription.PlanID)
	if err != nil {
		return fmt.Errorf("get subscription plan: %w", err)
	}

	eventID := uuid.New()
	payload, err := json.Marshal(subscriptionUpdatedEvent{
		EventID:        eventID.String(),
		SubscriptionID: subscription.ID.String(),
		StoreID:        subscription.StoreID.String(),
		PlanType:       plan.Type,
		Status:         subscription.Status,
		StartedAt:      subscription.StartedAt,
		ExpiresAt:      subscription.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("marshal subscription event: %w", err)
	}

	if err := outbox.Create(
		ctx,
		eventID.String(),
		subscription.ID.String(),
		subscriptionUpdatedEventType,
		payload,
	); err != nil {
		return fmt.Errorf("create subscription outbox event: %w", err)
	}
	return nil
}
