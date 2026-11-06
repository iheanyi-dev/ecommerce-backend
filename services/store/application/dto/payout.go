package dto

import "github.com/google/uuid"

// InitiatePayoutOutput contains the payout and provider-attempt details
// returned by the Order service.
type InitiatePayoutOutput struct {
	PayoutID      uuid.UUID
	AttemptID     uuid.UUID
	Reference     string
	Status        string
	AttemptStatus string
	Provider      string
	AmountMinor   int64
	Currency      string
}
