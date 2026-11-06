package dto

import "github.com/google/uuid"

// UpdateAfterSubscriptionInput contains a trusted subscription update
// received from the Billing service.
type UpdateAfterSubscriptionInput struct {
	EventID        uuid.UUID
	SubscriptionID uuid.UUID
	StoreID        uuid.UUID
	PlanType       string
	Status         string
}
