package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockSubscribeService struct {
	executeFunc func(context.Context, dto.SubscribeInput) (dto.SubscribeOutput, error)
}

func (m *mockSubscribeService) Execute(
	ctx context.Context,
	input dto.SubscribeInput,
) (dto.SubscribeOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestSubscribeHandler_Success(t *testing.T) {
	storeID := uuid.New()
	planID := uuid.New()
	subscriptionID := uuid.New()
	paymentID := uuid.New()

	handler := handlers.NewSubscribeHandler(&mockSubscribeService{
		executeFunc: func(ctx context.Context, input dto.SubscribeInput) (dto.SubscribeOutput, error) {
			if input.StoreID != storeID || input.PlanID != planID {
				t.Fatalf("unexpected input: %+v", input)
			}

			return dto.SubscribeOutput{
				SubscriptionID: subscriptionID,
				PaymentID:      paymentID,
				CheckoutURL:    "https://checkout.example.com/payment",
			}, nil
		},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/"+storeID.String()+"/subscribe",
		strings.NewReader(`{"plan_id":"`+planID.String()+`"}`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		SubscriptionID uuid.UUID `json:"subscription_id"`
		PaymentID      uuid.UUID `json:"payment_id"`
		CheckoutURL    string    `json:"checkout_url"`
	}

	if err := decodeTestJSON(rec, &response); err != nil {
		t.Fatal(err)
	}

	if response.SubscriptionID != subscriptionID {
		t.Fatalf("expected subscription ID %s, got %s", subscriptionID, response.SubscriptionID)
	}
	if response.PaymentID != paymentID {
		t.Fatalf("expected payment ID %s, got %s", paymentID, response.PaymentID)
	}
	if response.CheckoutURL != "https://checkout.example.com/payment" {
		t.Fatalf("unexpected checkout URL: %s", response.CheckoutURL)
	}
}

func TestSubscribeHandler_InvalidStoreID(t *testing.T) {
	handler := handlers.NewSubscribeHandler(&mockSubscribeService{
		executeFunc: func(context.Context, dto.SubscribeInput) (dto.SubscribeOutput, error) {
			t.Fatal("service must not be called")
			return dto.SubscribeOutput{}, nil
		},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/invalid/subscribe",
		strings.NewReader(`{"plan_id":"`+uuid.NewString()+`"}`),
	)
	req.SetPathValue("id", "invalid")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

func TestSubscribeHandler_InvalidJSON(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewSubscribeHandler(&mockSubscribeService{
		executeFunc: func(context.Context, dto.SubscribeInput) (dto.SubscribeOutput, error) {
			t.Fatal("service must not be called")
			return dto.SubscribeOutput{}, nil
		},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/"+storeID.String()+"/subscribe",
		strings.NewReader(`{`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

func TestSubscribeHandler_InvalidPlanID(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewSubscribeHandler(&mockSubscribeService{
		executeFunc: func(context.Context, dto.SubscribeInput) (dto.SubscribeOutput, error) {
			t.Fatal("service must not be called")
			return dto.SubscribeOutput{}, nil
		},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/"+storeID.String()+"/subscribe",
		strings.NewReader(`{"plan_id":"invalid"}`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

func TestSubscribeHandler_MissingPlanID(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewSubscribeHandler(&mockSubscribeService{
		executeFunc: func(context.Context, dto.SubscribeInput) (dto.SubscribeOutput, error) {
			t.Fatal("service must not be called")
			return dto.SubscribeOutput{}, nil
		},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/"+storeID.String()+"/subscribe",
		strings.NewReader(`{}`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

func TestSubscribeHandler_ServiceError(t *testing.T) {
	storeID := uuid.New()
	planID := uuid.New()

	handler := handlers.NewSubscribeHandler(&mockSubscribeService{
		executeFunc: func(context.Context, dto.SubscribeInput) (dto.SubscribeOutput, error) {
			return dto.SubscribeOutput{}, errors.New("billing unavailable")
		},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/"+storeID.String()+"/subscribe",
		strings.NewReader(`{"plan_id":"`+planID.String()+`"}`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}

func TestSubscribeHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewSubscribeHandler(&mockSubscribeService{
		executeFunc: func(context.Context, dto.SubscribeInput) (dto.SubscribeOutput, error) {
			t.Fatal("service must not be called")
			return dto.SubscribeOutput{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+uuid.NewString()+"/subscribe", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusMethodNotAllowed, "method not allowed")
}

var _ ports.SubscribeService = (*mockSubscribeService)(nil)
