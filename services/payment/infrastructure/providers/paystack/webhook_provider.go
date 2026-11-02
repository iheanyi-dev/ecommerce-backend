package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
)

const maxWebhookBodyBytes = 1 << 20

type WebhookProvider struct {
	secretKey string
}

func NewWebhookProvider(client *Client) (*WebhookProvider, error) {
	if client == nil || strings.TrimSpace(client.secretKey) == "" {
		return nil, ErrInvalidConfiguration
	}

	return &WebhookProvider{secretKey: client.secretKey}, nil
}

func (p *WebhookProvider) VerifyWebhook(
	ctx context.Context,
	body []byte,
	headers map[string]string,
) (ports.ProviderWebhookEvent, error) {
	if p == nil || strings.TrimSpace(p.secretKey) == "" {
		return ports.ProviderWebhookEvent{}, ErrInvalidConfiguration
	}
	if err := ctx.Err(); err != nil {
		return ports.ProviderWebhookEvent{}, err
	}
	if len(body) == 0 || len(body) > maxWebhookBodyBytes {
		return ports.ProviderWebhookEvent{}, ErrInvalidResponse
	}

	signature := webhookSignature(headers)
	if signature == "" || !p.validSignature(body, signature) {
		return ports.ProviderWebhookEvent{}, ports.ErrInvalidWebhookSignature
	}

	var payload struct {
		Event string `json:"event"`
		Data  struct {
			ID        json.Number     `json:"id"`
			Reference string          `json:"reference"`
			Metadata  json.RawMessage `json:"metadata"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return ports.ProviderWebhookEvent{}, fmt.Errorf("%w: decode webhook: %v", ErrInvalidResponse, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return ports.ProviderWebhookEvent{}, ErrInvalidResponse
	}

	eventType := strings.TrimSpace(payload.Event)
	if eventType == "" || payload.Data.ID == "" {
		return ports.ProviderWebhookEvent{}, ErrInvalidResponse
	}

	resourceType := ""
	resourceID := ""

	switch {
	case strings.HasPrefix(eventType, "charge."):
		resourceType = ports.WebhookResourcePayment
		resourceID = metadataID(payload.Data.Metadata, "payment_id")

	case strings.HasPrefix(eventType, "transfer."):
		resourceType = ports.WebhookResourcePayout
		resourceID = metadataID(payload.Data.Metadata, "payout_id")

	default:
		return ports.ProviderWebhookEvent{}, ErrInvalidResponse
	}

	if resourceID == "" {
		return ports.ProviderWebhookEvent{}, ErrInvalidResponse
	}

	eventID := strings.TrimSpace(payload.Data.ID.String())
	if eventID == "" {
		return ports.ProviderWebhookEvent{}, ErrInvalidResponse
	}

	return ports.ProviderWebhookEvent{
		EventID:      eventID,
		EventType:    eventType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}, nil
}

func (p *WebhookProvider) validSignature(body []byte, signature string) bool {
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return false
	}

	mac := hmac.New(sha512.New, []byte(p.secretKey))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func webhookSignature(headers map[string]string) string {
	for key, value := range headers {
		if strings.EqualFold(strings.TrimSpace(key), "x-paystack-signature") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func metadataID(raw json.RawMessage, key string) string {
	if len(raw) == 0 || !json.Valid(raw) {
		return ""
	}

	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return ""
	}

	var value string
	if err := json.Unmarshal(metadata[key], &value); err == nil {
		return strings.TrimSpace(value)
	}

	var numeric json.Number
	if err := json.Unmarshal(metadata[key], &numeric); err == nil {
		return strings.TrimSpace(numeric.String())
	}

	return ""
}
