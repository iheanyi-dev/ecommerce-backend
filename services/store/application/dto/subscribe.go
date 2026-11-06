package dto

import "github.com/google/uuid"

// SubscribeInput contains the Store and Billing plan identifiers required
// to initiate a subscription checkout.
type SubscribeInput struct {
	StoreID uuid.UUID
	PlanID  uuid.UUID
}

// SubscribeOutput contains the checkout details returned to the Store owner.
type SubscribeOutput struct {
	SubscriptionID uuid.UUID
	PaymentID      uuid.UUID
	CheckoutURL    string
}
