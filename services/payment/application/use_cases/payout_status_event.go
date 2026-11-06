package use_cases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
)

const payoutStatusChangedEventType = "payout.status_changed"

type payoutStatusChangedPayload struct {
	EventID               string             `json:"event_id"`
	PayoutID              string             `json:"payout_id"`
	AttemptID             string             `json:"attempt_id"`
	OrderID               string             `json:"order_id"`
	StoreID               string             `json:"store_id"`
	BusinessReferenceType string             `json:"business_reference_type"`
	BusinessReference     string             `json:"business_reference"`
	Status                enums.PayoutStatus `json:"status"`
}

// enqueuePayoutStatusChanged stores a terminal payout event in the caller's
// transaction. Each payout attempt has a distinct event identity.
func enqueuePayoutStatusChanged(
	ctx context.Context,
	outbox ports.PaymentOutboxRepository,
	payout *entities.Payout,
) error {
	if payout.Status() != enums.PayoutStatusSucceeded &&
		payout.Status() != enums.PayoutStatusFailed {
		return nil
	}

	eventID := uuid.NewString()
	attemptID := payout.AttemptID()

	payload, err := json.Marshal(payoutStatusChangedPayload{
		EventID:               eventID,
		PayoutID:              payout.ID(),
		AttemptID:             attemptID,
		OrderID:               payout.OrderID(),
		StoreID:               payout.StoreID(),
		BusinessReferenceType: payout.BusinessReference().Type(),
		BusinessReference:     payout.BusinessReference().ID(),
		Status:                payout.Status(),
	})
	if err != nil {
		return fmt.Errorf("marshal payout status event: %w", err)
	}

	deduplicationKey := payout.ID() + ":" + attemptID + ":" + payoutStatusChangedEventType
	if err := outbox.Create(
		ctx,
		eventID,
		"payout",
		payout.ID(),
		deduplicationKey,
		payoutStatusChangedEventType,
		payload,
	); err != nil {
		return fmt.Errorf("enqueue payout status event: %w", err)
	}

	return nil
}
