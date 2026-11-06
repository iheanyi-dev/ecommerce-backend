package schemas

import (
	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
)

// SubscribeRequest contains the Billing plan requested by the Store owner.
type SubscribeRequest struct {
	PlanID uuid.UUID `json:"plan_id"`
}

// SubscribeResponse contains the subscription checkout details.
type SubscribeResponse struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	PaymentID      uuid.UUID `json:"payment_id"`
	CheckoutURL    string    `json:"checkout_url"`
}

// NewSubscribeResponse converts the application output into an HTTP response.
func NewSubscribeResponse(output dto.SubscribeOutput) SubscribeResponse {
	return SubscribeResponse{
		SubscriptionID: output.SubscriptionID,
		PaymentID:      output.PaymentID,
		CheckoutURL:    output.CheckoutURL,
	}
}
