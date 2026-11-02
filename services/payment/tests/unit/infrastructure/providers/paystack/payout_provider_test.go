package paystack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/providers/paystack"
)

func newPayoutFixture(t *testing.T) (*entities.Payout, *entities.PayoutAttempt) {
	t.Helper()

	businessReference, err := value_objects.NewBusinessReference("order", "order-123")
	if err != nil {
		t.Fatal(err)
	}
	amount, err := value_objects.NewMoney(250000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	payout, err := entities.NewPayout(
		businessReference,
		"store-123",
		"order-123",
		"Vendor settlement",
		amount,
		entities.PayoutBeneficiary{
			AccountName:   "Test Vendor",
			AccountNumber: "0123456789",
			BankCode:      "058",
			BankName:      "Test Bank",
			CountryCode:   "NG",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := entities.NewPayoutAttempt(payout.ID(), "paystack")
	if err != nil {
		t.Fatal(err)
	}
	if err := payout.SetAttemptID(attempt.ID()); err != nil {
		t.Fatal(err)
	}
	return payout, attempt
}

func TestPayoutProviderInitiatesTransfer(t *testing.T) {
	payout, attempt := newPayoutFixture(t)
	recipientCreated := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/transferrecipient":
			if r.Method != http.MethodPost {
				t.Fatalf("recipient method = %s", r.Method)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["type"] != "nuban" ||
				body["account_number"] != "0123456789" ||
				body["bank_code"] != "058" ||
				body["currency"] != "NGN" {
				t.Fatalf("unexpected recipient payload: %+v", body)
			}
			recipientCreated = true
			_, _ = w.Write([]byte(`{"status":true,"data":{"recipient_code":"RCP_test123"}}`))

		case "/transfer":
			if !recipientCreated {
				t.Fatal("transfer initiated before recipient creation")
			}
			var body struct {
				Amount    int64           `json:"amount"`
				Source    string          `json:"source"`
				Recipient string          `json:"recipient"`
				Reference string          `json:"reference"`
				Metadata  json.RawMessage `json:"metadata"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Amount != 250000 || body.Source != "balance" ||
				body.Recipient != "RCP_test123" || body.Reference != attempt.ID() {
				t.Fatalf("unexpected transfer payload: %+v", body)
			}

			var metadata map[string]string
			if err := json.Unmarshal(body.Metadata, &metadata); err != nil {
				t.Fatal(err)
			}
			if metadata["payout_id"] != payout.ID() ||
				metadata["payout_attempt_id"] != attempt.ID() {
				t.Fatalf("unexpected transfer metadata: %+v", metadata)
			}

			_, _ = w.Write([]byte(`{"status":true,"data":{"reference":"` + attempt.ID() + `","status":"pending","transfer_code":"TRF_test123"}}`))

		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPayoutProvider(client)
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.InitiatePayout(context.Background(), ports.PayoutInitiationRequest{
		Payout: payout, Attempt: attempt,
	})
	if err != nil {
		t.Fatalf("InitiatePayout() error = %v", err)
	}
	if result.ProviderReference != attempt.ID() ||
		result.Status != enums.PayoutAttemptStatusProcessing {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPayoutProviderRejectsUnsupportedCurrency(t *testing.T) {
	payout, _ := newPayoutFixture(t)
	amount, err := value_objects.NewMoney(100, "USD")
	if err != nil {
		t.Fatal(err)
	}

	unsupported, err := entities.NewPayout(
		payout.BusinessReference(),
		payout.StoreID(),
		payout.OrderID(),
		payout.Reason(),
		amount,
		payout.Beneficiary(),
	)
	if err != nil {
		t.Fatal(err)
	}
	unsupportedAttempt, err := entities.NewPayoutAttempt(unsupported.ID(), "paystack")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unsupported payout must not call Paystack")
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPayoutProvider(client)
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider.InitiatePayout(context.Background(), ports.PayoutInitiationRequest{
		Payout: unsupported, Attempt: unsupportedAttempt,
	})
	if err == nil {
		t.Fatal("expected unsupported currency error")
	}
}

func TestPayoutProviderReconcilesSuccessfulTransfer(t *testing.T) {
	payout, attempt := newPayoutFixture(t)
	if err := attempt.SetProviderReference("transfer-ref-123"); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/transfer/verify/transfer-ref-123" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":true,"data":{"amount":250000,"currency":"NGN","reference":"transfer-ref-123","status":"success","reason":""}}`))
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPayoutProvider(client)
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.ReconcilePayout(context.Background(), ports.PayoutReconciliationRequest{
		Payout: payout, Attempt: attempt,
	})
	if err != nil {
		t.Fatalf("ReconcilePayout() error = %v", err)
	}
	if result.Amount != 250000 || result.Currency != "NGN" ||
		result.ProviderReference != "transfer-ref-123" ||
		result.Status != enums.PayoutAttemptStatusSucceeded {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPayoutProviderReconcilesFailedTransfer(t *testing.T) {
	payout, attempt := newPayoutFixture(t)
	if err := attempt.SetProviderReference("transfer-ref-456"); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":true,"data":{"amount":250000,"currency":"NGN","reference":"transfer-ref-456","status":"failed","reason":"Insufficient balance"}}`))
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPayoutProvider(client)
	if err != nil {
		t.Fatal(err)
	}

	result, err := provider.ReconcilePayout(context.Background(), ports.PayoutReconciliationRequest{
		Payout: payout, Attempt: attempt,
	})
	if err != nil {
		t.Fatalf("ReconcilePayout() error = %v", err)
	}
	if result.Status != enums.PayoutAttemptStatusFailed ||
		result.FailureReason != "Insufficient balance" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPayoutProviderRejectsMismatchedTransferReference(t *testing.T) {
	payout, attempt := newPayoutFixture(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/transferrecipient":
			_, _ = w.Write([]byte(`{"status":true,"data":{"recipient_code":"RCP_test123"}}`))
		case "/transfer":
			_, _ = w.Write([]byte(`{"status":true,"data":{"reference":"different-reference","status":"pending"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := paystack.NewClient("test-secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := paystack.NewPayoutProvider(client)
	if err != nil {
		t.Fatal(err)
	}

	_, err = provider.InitiatePayout(context.Background(), ports.PayoutInitiationRequest{
		Payout: payout, Attempt: attempt,
	})
	if err != paystack.ErrInvalidResponse {
		t.Fatalf("error = %v; want ErrInvalidResponse", err)
	}
}
