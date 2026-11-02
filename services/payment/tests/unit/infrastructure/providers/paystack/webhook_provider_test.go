package paystack_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/providers/paystack"
)

const webhookSecret = "test-paystack-secret"

func signWebhook(body []byte) string {
	mac := hmac.New(sha512.New, []byte(webhookSecret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func newWebhookProvider(t *testing.T) *paystack.WebhookProvider {
	t.Helper()

	client, err := paystack.NewClient(webhookSecret, "https://api.paystack.co", nil)
	if err != nil {
		t.Fatal(err)
	}

	provider, err := paystack.NewWebhookProvider(client)
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestWebhookProviderVerifiesChargeEvent(t *testing.T) {
	body := []byte(`{
		"event":"charge.success",
		"data":{
			"id":12345,
			"reference":"payment-reference",
			"metadata":{"payment_id":"11111111-1111-4111-8111-111111111111"}
		}
	}`)

	provider := newWebhookProvider(t)
	event, err := provider.VerifyWebhook(context.Background(), body, map[string]string{
		"X-Paystack-Signature": signWebhook(body),
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() error = %v", err)
	}

	if event.EventID != "12345" ||
		event.EventType != "charge.success" ||
		event.ResourceType != ports.WebhookResourcePayment ||
		event.ResourceID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestWebhookProviderVerifiesTransferEvent(t *testing.T) {
	body := []byte(`{
		"event":"transfer.success",
		"data":{
			"id":67890,
			"reference":"payout-reference",
			"metadata":{"payout_id":"22222222-2222-4222-8222-222222222222"}
		}
	}`)

	provider := newWebhookProvider(t)
	event, err := provider.VerifyWebhook(context.Background(), body, map[string]string{
		"x-paystack-signature": signWebhook(body),
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() error = %v", err)
	}

	if event.EventID != "67890" ||
		event.ResourceType != ports.WebhookResourcePayout ||
		event.ResourceID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestWebhookProviderRejectsInvalidSignature(t *testing.T) {
	body := []byte(`{"event":"charge.success","data":{"id":123,"metadata":{"payment_id":"11111111-1111-4111-8111-111111111111"}}}`)

	_, err := newWebhookProvider(t).VerifyWebhook(
		context.Background(),
		body,
		map[string]string{"x-paystack-signature": "invalid"},
	)
	if !errors.Is(err, ports.ErrInvalidWebhookSignature) {
		t.Fatalf("error = %v; want ErrInvalidWebhookSignature", err)
	}
}

func TestWebhookProviderRejectsMissingSignature(t *testing.T) {
	body := []byte(`{"event":"charge.success","data":{"id":123}}`)

	_, err := newWebhookProvider(t).VerifyWebhook(context.Background(), body, nil)
	if !errors.Is(err, ports.ErrInvalidWebhookSignature) {
		t.Fatalf("error = %v; want ErrInvalidWebhookSignature", err)
	}
}

func TestWebhookProviderRejectsMissingInternalID(t *testing.T) {
	body := []byte(`{"event":"charge.success","data":{"id":123,"reference":"some-reference","metadata":{}}}`)

	_, err := newWebhookProvider(t).VerifyWebhook(context.Background(), body, map[string]string{
		"x-paystack-signature": signWebhook(body),
	})
	if err == nil {
		t.Fatal("expected missing internal ID error")
	}
}

func TestWebhookProviderRejectsUnsupportedEvent(t *testing.T) {
	body := []byte(`{"event":"customer.create","data":{"id":123}}`)

	_, err := newWebhookProvider(t).VerifyWebhook(context.Background(), body, map[string]string{
		"x-paystack-signature": signWebhook(body),
	})
	if err == nil {
		t.Fatal("expected unsupported event error")
	}
}

func TestWebhookProviderRejectsMalformedPayload(t *testing.T) {
	body := []byte(`{"event":`)

	_, err := newWebhookProvider(t).VerifyWebhook(context.Background(), body, map[string]string{
		"x-paystack-signature": signWebhook(body),
	})
	if err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestWebhookProviderRejectsTamperedBody(t *testing.T) {
	body := []byte(`{"event":"charge.success","data":{"id":123,"metadata":{"payment_id":"11111111-1111-4111-8111-111111111111"}}}`)
	signature := signWebhook(body)
	tampered := append([]byte(nil), body...)
	tampered[len(tampered)-2] = ' '

	_, err := newWebhookProvider(t).VerifyWebhook(context.Background(), tampered, map[string]string{
		"x-paystack-signature": signature,
	})
	if !errors.Is(err, ports.ErrInvalidWebhookSignature) {
		t.Fatalf("error = %v; want invalid signature", err)
	}
}

func TestWebhookProviderRejectsCancelledContext(t *testing.T) {
	body := []byte(`{"event":"charge.success","data":{"id":123,"metadata":{"payment_id":"11111111-1111-4111-8111-111111111111"}}}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newWebhookProvider(t).VerifyWebhook(ctx, body, map[string]string{
		"x-paystack-signature": signWebhook(body),
	})
	if err == nil {
		t.Fatal("expected context cancellation")
	}
}
