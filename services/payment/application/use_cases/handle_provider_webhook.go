package use_cases

import (
	"context"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
)

type paymentWebhookVerifier interface {
	Execute(context.Context, string) (*PaymentDetails, error)
}

type payoutWebhookVerifier interface {
	Execute(context.Context, string) (*InitiatePayoutOutput, error)
}

type ProviderWebhookOutput struct {
	EventID      string `json:"event_id"`
	EventType    string `json:"event_type"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

// HandleProviderWebhookUseCase authenticates a provider event, then delegates
// reconciliation and persistence to the existing verification use cases.
type HandleProviderWebhookUseCase struct {
	provider      ports.WebhookProvider
	verifyPayment paymentWebhookVerifier
	verifyPayout  payoutWebhookVerifier
}

func NewHandleProviderWebhookUseCase(
	provider ports.WebhookProvider,
	verifyPayment paymentWebhookVerifier,
	verifyPayout payoutWebhookVerifier,
) (*HandleProviderWebhookUseCase, error) {
	if provider == nil || verifyPayment == nil || verifyPayout == nil {
		return nil, fmt.Errorf("provider webhook dependencies are required")
	}

	return &HandleProviderWebhookUseCase{
		provider:      provider,
		verifyPayment: verifyPayment,
		verifyPayout:  verifyPayout,
	}, nil
}

func (uc *HandleProviderWebhookUseCase) Execute(
	ctx context.Context,
	body []byte,
	headers map[string]string,
) (*ProviderWebhookOutput, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty provider webhook body")
	}

	event, err := uc.provider.VerifyWebhook(ctx, body, headers)
	if err != nil {
		return nil, fmt.Errorf("verify provider webhook: %w", err)
	}

	event.EventID = strings.TrimSpace(event.EventID)
	event.EventType = strings.TrimSpace(event.EventType)
	event.ResourceType = strings.TrimSpace(event.ResourceType)
	event.ResourceID = strings.TrimSpace(event.ResourceID)

	if event.EventID == "" || event.EventType == "" {
		return nil, application.ErrInvalidProviderResponse
	}
	if err := application.ValidateUUID(event.ResourceID); err != nil {
		return nil, fmt.Errorf("webhook resource ID: %w", err)
	}

	switch event.ResourceType {
	case ports.WebhookResourcePayment:
		if _, err := uc.verifyPayment.Execute(ctx, event.ResourceID); err != nil {
			return nil, fmt.Errorf("verify webhook payment: %w", err)
		}
	case ports.WebhookResourcePayout:
		if _, err := uc.verifyPayout.Execute(ctx, event.ResourceID); err != nil {
			return nil, fmt.Errorf("verify webhook payout: %w", err)
		}
	default:
		return nil, application.ErrInvalidProviderResponse
	}

	return &ProviderWebhookOutput{
		EventID:      event.EventID,
		EventType:    event.EventType,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
	}, nil
}
