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

const paymentStatusChangedEventType = "payment.status_changed"

type paymentStatusChangedPayload struct {
	EventID               string              `json:"event_id"`
	BusinessReferenceType string              `json:"business_reference_type"`
	BusinessReference     string              `json:"business_reference"`
	Status                enums.PaymentStatus `json:"status"`
}

// enqueuePaymentStatusChanged stores a terminal payment event in the caller's
// transaction. Nonterminal payment states do not produce business events.
func enqueuePaymentStatusChanged(
	ctx context.Context,
	outbox ports.PaymentOutboxRepository,
	payment *entities.Payment,
) error {
	if payment.Status() != enums.PaymentStatusSucceeded &&
		payment.Status() != enums.PaymentStatusFailed {
		return nil
	}

	eventID := uuid.NewString()
	payload, err := json.Marshal(paymentStatusChangedPayload{
		EventID:               eventID,
		BusinessReferenceType: payment.BusinessReference().Type(),
		BusinessReference:     payment.BusinessReference().ID(),
		Status:                payment.Status(),
	})
	if err != nil {
		return fmt.Errorf("marshal payment status event: %w", err)
	}

	if err := outbox.Create(
		ctx,
		eventID,
		"payment",
		payment.ID(),
		payment.ID()+":"+paymentStatusChangedEventType,
		paymentStatusChangedEventType,
		payload,
	); err != nil {
		return fmt.Errorf("enqueue payment status event: %w", err)
	}
	return nil
}
