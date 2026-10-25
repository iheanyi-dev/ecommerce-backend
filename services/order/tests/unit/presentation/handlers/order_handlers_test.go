package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/presentation/schemas"
)

func newHandlers() *handlers.OrderHandlers {
	return handlers.NewOrderHandlers(nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestCreateOrderRejectsInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: "{"},
		{name: "missing cart ID", body: `{"delivery_location":"Lagos"}`},
		{name: "unknown field", body: `{"cart_id":"c5b0d0a7-48f1-4f3b-b9a6-39e8c1d3b7a2","delivery_location":"Lagos","unexpected":true}`},
		{name: "trailing JSON", body: `{"cart_id":"c5b0d0a7-48f1-4f3b-b9a6-39e8c1d3b7a2","delivery_location":"Lagos"} {}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(tt.body))
			req = req.WithContext(ports.WithAuthenticatedIdentity(req.Context(), ports.AuthenticatedIdentity{UserID: uuid.New(), Role: "customer"}))
			rec := httptest.NewRecorder()
			newHandlers().CreateOrder(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateOrderRejectsWrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	rec := httptest.NewRecorder()
	newHandlers().CreateOrder(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestRequestValidation(t *testing.T) {
	if (schemas.CreateOrderRequest{}).Validate() {
		t.Fatal("empty create request should be invalid")
	}
	if (schemas.AddShippingRequest{ShippingAmount: -1}).Validate() {
		t.Fatal("negative shipping amount should be invalid")
	}
	if !(schemas.AddShippingRequest{ShippingAmount: 0}).Validate() {
		t.Fatal("zero shipping amount should be valid")
	}
}
