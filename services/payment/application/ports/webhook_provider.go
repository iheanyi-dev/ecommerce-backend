package ports

import (
	"context"
	"errors"
)

var ErrInvalidWebhookSignature = errors.New("invalid provider webhook signature")

const (
	WebhookResourcePayment = "payment"
	WebhookResourcePayout  = "payout"
)

// ProviderWebhookEvent is a provider-neutral event whose signature and payload
// have already been verified by the provider adapter.
//
// ResourceID must be our internal payment or payout UUID, obtained from trusted
// provider metadata or an unambiguous provider-reference lookup.
type ProviderWebhookEvent struct {
	EventID      string
	EventType    string
	ResourceType string
	ResourceID   string
}

// WebhookProvider verifies the provider signature and normalizes its event.
// Implementations must reject invalid signatures and malformed payloads.
type WebhookProvider interface {
	VerifyWebhook(
		ctx context.Context,
		body []byte,
		headers map[string]string,
	) (ProviderWebhookEvent, error)
}
