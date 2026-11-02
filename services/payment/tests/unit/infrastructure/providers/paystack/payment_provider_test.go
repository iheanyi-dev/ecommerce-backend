package paystack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/providers/paystack"
)

func TestPaymentProviderInitiateCheckout(t *testing.T) {
	var received struct {
		Email     string          `json:"email"`
		Amount    int64           `json:"amount"`
		Currency  string          `json:"currency"`
		Reference string          `json:"reference"`
		Metadata  json.RawMessage `json:"metadata"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/transaction/initialize" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"success","data":{"authorization_url":"https://checkout.paystack.com/example","access_code":"access-1","reference":"attempt-id-1"}}`))
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPaymentProvider(client)
	if err != nil {
		t.Fatal(err)
	}
	amount, err := value_objects.NewMoney(12500, "NGN")
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.InitiateCheckout(context.Background(), ports.CheckoutRequest{
		PaymentID:        "payment-id-1",
		AttemptID:        "attempt-id-1",
		PaymentReference: "payment-ref-1",
		CustomerEmail:    "customer@example.com",
		Amount:           amount,
	})
	if err != nil {
		t.Fatalf("InitiateCheckout() error = %v", err)
	}

	if received.Amount != 12500 {
		t.Fatalf("amount = %d; want 12500 minor units", received.Amount)
	}
	if received.Email != "customer@example.com" ||
		received.Currency != "NGN" ||
		received.Reference != "attempt-id-1" {
		t.Fatalf("unexpected payload: %+v", received)
	}

	var metadata map[string]string
	if err := json.Unmarshal(received.Metadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["payment_id"] != "payment-id-1" ||
		metadata["payment_attempt_id"] != "attempt-id-1" {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}

	if result.ProviderReference != "attempt-id-1" ||
		result.CheckoutURL != "https://checkout.paystack.com/example" ||
		result.Status != enums.PaymentAttemptStatusPending {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPaymentProviderRejectsMismatchedInitializationReference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":true,"data":{"authorization_url":"https://checkout.paystack.com/example","reference":"different-reference"}}`))
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPaymentProvider(client)
	if err != nil {
		t.Fatal(err)
	}
	amount, _ := value_objects.NewMoney(100, "NGN")

	_, err = provider.InitiateCheckout(context.Background(), ports.CheckoutRequest{
		PaymentID:        "payment-id",
		AttemptID:        "attempt-id",
		PaymentReference: "expected-reference",
		CustomerEmail:    "customer@example.com",
		Amount:           amount,
	})
	if err != paystack.ErrInvalidResponse {
		t.Fatalf("error = %v; want ErrInvalidResponse", err)
	}
}

func TestPaymentProviderRejectsInvalidCheckoutInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request should not reach Paystack")
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPaymentProvider(client)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := provider.InitiateCheckout(context.Background(), ports.CheckoutRequest{}); err == nil {
		t.Fatal("expected invalid checkout request error")
	}
}
