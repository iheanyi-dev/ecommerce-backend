package paystack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/providers/paystack"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*paystack.Client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)
	client, err := paystack.NewClient("sk_test_secret", server.URL, server.Client())
	if err != nil {
		server.Close()
		t.Fatalf("NewClient() error = %v", err)
	}

	t.Cleanup(server.Close)
	return client, server
}

func TestNewClientRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		baseURL   string
	}{
		{name: "empty secret", baseURL: "https://api.paystack.co"},
		{name: "invalid scheme", secretKey: "sk_test", baseURL: "ftp://api.paystack.co"},
		{name: "missing host", secretKey: "sk_test", baseURL: "https:///"},
		{name: "URL credentials", secretKey: "sk_test", baseURL: "https://user:pass@example.com"},
		{name: "query string", secretKey: "sk_test", baseURL: "https://example.com?key=value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := paystack.NewClient(tt.secretKey, tt.baseURL, nil)
			if !errors.Is(err, paystack.ErrInvalidConfiguration) {
				t.Errorf("error = %v, want ErrInvalidConfiguration", err)
			}
		})
	}
}

func TestClientRequestSendsAuthenticatedJSONRequest(t *testing.T) {
	type requestPayload struct {
		Email  string `json:"email"`
		Amount int64  `json:"amount"`
	}
	type responseData struct {
		Reference string `json:"reference"`
	}

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/transaction/initialize" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_secret" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}

		var payload requestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if payload.Email != "customer@example.com" || payload.Amount != 12500 {
			t.Errorf("unexpected request payload: %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"Success","data":{"reference":"ref-123"}}`))
	})

	var got responseData
	err := client.Request(
		context.Background(),
		http.MethodPost,
		"/transaction/initialize",
		requestPayload{Email: "customer@example.com", Amount: 12500},
		&got,
	)
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if got.Reference != "ref-123" {
		t.Errorf("reference = %q, want ref-123", got.Reference)
	}
}

func TestClientRequestReturnsProviderErrorForHTTPFailure(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":false,"message":"Invalid key"}`))
	})

	err := client.Request(context.Background(), http.MethodGet, "/transaction/verify/ref", nil, nil)

	var providerErr *paystack.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d", providerErr.StatusCode)
	}
	if providerErr.Message != "Invalid key" {
		t.Errorf("message = %q", providerErr.Message)
	}
	if !errors.Is(err, paystack.ErrAPIRequestFailed) {
		t.Error("error should unwrap to ErrAPIRequestFailed")
	}
}

func TestClientRequestReturnsProviderErrorForUnsuccessfulEnvelope(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":false,"message":"Transaction not found"}`))
	})

	err := client.Request(context.Background(), http.MethodGet, "/transaction/verify/ref", nil, nil)

	var providerErr *paystack.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want ProviderError", err)
	}
	if providerErr.Message != "Transaction not found" {
		t.Errorf("message = %q", providerErr.Message)
	}
}

func TestClientRequestRejectsInvalidResponse(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	})

	err := client.Request(context.Background(), http.MethodGet, "/transaction/verify/ref", nil, nil)
	if !errors.Is(err, paystack.ErrInvalidResponse) {
		t.Errorf("error = %v, want ErrInvalidResponse", err)
	}
}

func TestClientRequestRejectsOversizedResponse(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":"` +
			strings.Repeat("x", (1<<20)+1) + `"}`))
	})

	err := client.Request(context.Background(), http.MethodGet, "/transaction/verify/ref", nil, nil)
	if !errors.Is(err, paystack.ErrResponseTooLarge) {
		t.Errorf("error = %v, want ErrResponseTooLarge", err)
	}
}

func TestClientRequestRejectsAbsoluteRequestPath(t *testing.T) {
	client, _ := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("server must not be called")
	})

	err := client.Request(context.Background(), http.MethodGet, "https://evil.example/path", nil, nil)
	if !errors.Is(err, paystack.ErrInvalidRequestPath) {
		t.Errorf("error = %v, want ErrInvalidRequestPath", err)
	}
}
