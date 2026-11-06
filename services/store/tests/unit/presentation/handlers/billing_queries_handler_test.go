package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockListBillingPlansService struct {
	executeFunc func(context.Context) ([]dto.BillingPlanOutput, error)
}

func (m *mockListBillingPlansService) Execute(ctx context.Context) ([]dto.BillingPlanOutput, error) {
	return m.executeFunc(ctx)
}

type mockGetBillingPlanService struct {
	executeFunc func(context.Context, dto.GetBillingPlanInput) (dto.BillingPlanOutput, error)
}

func (m *mockGetBillingPlanService) Execute(
	ctx context.Context,
	input dto.GetBillingPlanInput,
) (dto.BillingPlanOutput, error) {
	return m.executeFunc(ctx, input)
}

type mockListStoreSubscriptionsService struct {
	executeFunc func(context.Context, dto.ListStoreSubscriptionsInput) ([]dto.BillingSubscriptionOutput, error)
}

func (m *mockListStoreSubscriptionsService) Execute(
	ctx context.Context,
	input dto.ListStoreSubscriptionsInput,
) ([]dto.BillingSubscriptionOutput, error) {
	return m.executeFunc(ctx, input)
}

type mockGetStoreSubscriptionService struct {
	executeFunc func(context.Context, dto.GetStoreSubscriptionInput) (dto.BillingSubscriptionOutput, error)
}

func (m *mockGetStoreSubscriptionService) Execute(
	ctx context.Context,
	input dto.GetStoreSubscriptionInput,
) (dto.BillingSubscriptionOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestListBillingPlansHandler_Success(t *testing.T) {
	planID := uuid.New()
	now := time.Now().UTC()
	handler := handlers.NewListBillingPlansHandler(&mockListBillingPlansService{
		executeFunc: func(context.Context) ([]dto.BillingPlanOutput, error) {
			return []dto.BillingPlanOutput{
				{
					ID:          planID,
					Type:        "premium",
					Name:        "Premium",
					Description: "Premium plan",
					Price:       dto.BillingMoneyOutput{AmountMinor: 500000, Currency: "NGN"},
					Duration:    30 * 24 * time.Hour,
					MaxProducts: 5,
					AIEnabled:   true,
					Available:   true,
					Promotion: &dto.BillingPromotionOutput{
						Name:                "Launch offer",
						DiscountAmountMinor: 50000,
						StartsAt:            &now,
						EndsAt:              &now,
					},
					CreatedAt: &now,
					UpdatedAt: &now,
				},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/plans", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Plans []struct {
			ID          uuid.UUID `json:"id"`
			Type        string    `json:"type"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
			Price       struct {
				AmountMinor int64  `json:"amount_minor"`
				Currency    string `json:"currency"`
			} `json:"price"`
			MaxProducts int  `json:"max_products"`
			AIEnabled   bool `json:"ai_enabled"`
			Available   bool `json:"available"`
			Promotion   *struct {
				Name                string `json:"name"`
				DiscountAmountMinor int64  `json:"discount_amount_minor"`
			} `json:"promotion"`
		} `json:"plans"`
	}
	if err := decodeTestJSON(rec, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Plans) != 1 {
		t.Fatalf("expected one plan, got %d", len(response.Plans))
	}

	plan := response.Plans[0]
	if plan.ID != planID || plan.Type != "premium" || plan.Name != "Premium" {
		t.Fatalf("unexpected plan identity: %+v", plan)
	}
	if plan.Description != "Premium plan" {
		t.Fatalf("unexpected description: %q", plan.Description)
	}
	if plan.Price.AmountMinor != 500000 || plan.Price.Currency != "NGN" {
		t.Fatalf("unexpected price: %+v", plan.Price)
	}
	if plan.MaxProducts != 5 || !plan.AIEnabled || !plan.Available {
		t.Fatalf("unexpected plan capabilities: %+v", plan)
	}
	if plan.Promotion == nil ||
		plan.Promotion.Name != "Launch offer" ||
		plan.Promotion.DiscountAmountMinor != 50000 {
		t.Fatalf("unexpected promotion: %+v", plan.Promotion)
	}
}

func TestListBillingPlansHandler_EmptyCatalogue(t *testing.T) {
	handler := handlers.NewListBillingPlansHandler(&mockListBillingPlansService{
		executeFunc: func(context.Context) ([]dto.BillingPlanOutput, error) {
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/plans", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	var response struct {
		Plans []json.RawMessage `json:"plans"`
	}
	if err := decodeTestJSON(rec, &response); err != nil {
		t.Fatal(err)
	}
	if response.Plans == nil || len(response.Plans) != 0 {
		t.Fatalf("expected an empty plans array, got %+v", response.Plans)
	}
}

func TestListBillingPlansHandler_ServiceError(t *testing.T) {
	handler := handlers.NewListBillingPlansHandler(&mockListBillingPlansService{
		executeFunc: func(context.Context) ([]dto.BillingPlanOutput, error) {
			return nil, errors.New("billing unavailable")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/plans", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}

func TestListBillingPlansHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewListBillingPlansHandler(&mockListBillingPlansService{
		executeFunc: func(context.Context) ([]dto.BillingPlanOutput, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/plans", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusMethodNotAllowed, "method not allowed")
}

func TestGetBillingPlanHandler_Success(t *testing.T) {
	planID := uuid.New()
	handler := handlers.NewGetBillingPlanHandler(&mockGetBillingPlanService{
		executeFunc: func(_ context.Context, input dto.GetBillingPlanInput) (dto.BillingPlanOutput, error) {
			if input.PlanID != planID {
				t.Fatalf("expected plan ID %s, got %s", planID, input.PlanID)
			}
			return dto.BillingPlanOutput{
				ID:          planID,
				Type:        "premium",
				Name:        "Premium",
				Price:       dto.BillingMoneyOutput{AmountMinor: 500000, Currency: "NGN"},
				Duration:    30 * 24 * time.Hour,
				MaxProducts: 5,
				AIEnabled:   true,
				Available:   true,
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/plans/"+planID.String(), nil)
	req.SetPathValue("id", planID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		ID          uuid.UUID `json:"id"`
		Type        string    `json:"type"`
		Name        string    `json:"name"`
		MaxProducts int       `json:"max_products"`
		AIEnabled   bool      `json:"ai_enabled"`
		Available   bool      `json:"available"`
	}
	if err := decodeTestJSON(rec, &response); err != nil {
		t.Fatal(err)
	}
	if response.ID != planID || response.Type != "premium" || response.Name != "Premium" {
		t.Fatalf("unexpected plan response: %+v", response)
	}
	if response.MaxProducts != 5 || !response.AIEnabled || !response.Available {
		t.Fatalf("unexpected plan capabilities: %+v", response)
	}
}

func TestGetBillingPlanHandler_InvalidPlanID(t *testing.T) {
	handler := handlers.NewGetBillingPlanHandler(&mockGetBillingPlanService{
		executeFunc: func(context.Context, dto.GetBillingPlanInput) (dto.BillingPlanOutput, error) {
			t.Fatal("service must not be called")
			return dto.BillingPlanOutput{}, nil
		},
	})

	for _, id := range []string{"invalid", uuid.Nil.String()} {
		t.Run(id, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/plans/"+id, nil)
			req.SetPathValue("id", id)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assertBadRequest(t, rec)
		})
	}
}

func TestGetBillingPlanHandler_NotFound(t *testing.T) {
	handler := handlers.NewGetBillingPlanHandler(&mockGetBillingPlanService{
		executeFunc: func(context.Context, dto.GetBillingPlanInput) (dto.BillingPlanOutput, error) {
			return dto.BillingPlanOutput{}, application_errors.ErrBillingPlanNotAvailable
		},
	})

	planID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/plans/"+planID.String(), nil)
	req.SetPathValue("id", planID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "billing plan not found")
}

func TestGetBillingPlanHandler_ServiceError(t *testing.T) {
	handler := handlers.NewGetBillingPlanHandler(&mockGetBillingPlanService{
		executeFunc: func(context.Context, dto.GetBillingPlanInput) (dto.BillingPlanOutput, error) {
			return dto.BillingPlanOutput{}, errors.New("billing unavailable")
		},
	})

	planID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/plans/"+planID.String(), nil)
	req.SetPathValue("id", planID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}

func TestGetBillingPlanHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewGetBillingPlanHandler(&mockGetBillingPlanService{
		executeFunc: func(context.Context, dto.GetBillingPlanInput) (dto.BillingPlanOutput, error) {
			t.Fatal("service must not be called")
			return dto.BillingPlanOutput{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/plans/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusMethodNotAllowed, "method not allowed")
}

func TestListStoreSubscriptionsHandler_Success(t *testing.T) {
	storeID := uuid.New()
	subscriptionID := uuid.New()
	planID := uuid.New()
	handler := handlers.NewListStoreSubscriptionsHandler(&mockListStoreSubscriptionsService{
		executeFunc: func(_ context.Context, input dto.ListStoreSubscriptionsInput) ([]dto.BillingSubscriptionOutput, error) {
			if input.StoreID != storeID {
				t.Fatalf("expected store ID %s, got %s", storeID, input.StoreID)
			}
			return []dto.BillingSubscriptionOutput{
				{
					ID:       subscriptionID,
					StoreID:  storeID,
					PlanID:   planID,
					Price:    dto.BillingMoneyOutput{AmountMinor: 500000, Currency: "NGN"},
					Duration: 30 * 24 * time.Hour,
					Status:   "active",
				},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeID.String()+"/billing/subscriptions", nil)
	req.SetPathValue("id", storeID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		Subscriptions []struct {
			ID      uuid.UUID `json:"id"`
			StoreID uuid.UUID `json:"store_id"`
			PlanID  uuid.UUID `json:"plan_id"`
			Status  string    `json:"status"`
			Price   struct {
				AmountMinor int64  `json:"amount_minor"`
				Currency    string `json:"currency"`
			} `json:"price"`
		} `json:"subscriptions"`
	}
	if err := decodeTestJSON(rec, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Subscriptions) != 1 {
		t.Fatalf("expected one subscription, got %d", len(response.Subscriptions))
	}

	subscription := response.Subscriptions[0]
	if subscription.ID != subscriptionID ||
		subscription.StoreID != storeID ||
		subscription.PlanID != planID ||
		subscription.Status != "active" {
		t.Fatalf("unexpected subscription: %+v", subscription)
	}
	if subscription.Price.AmountMinor != 500000 || subscription.Price.Currency != "NGN" {
		t.Fatalf("unexpected subscription price: %+v", subscription.Price)
	}
}

func TestListStoreSubscriptionsHandler_InvalidStoreID(t *testing.T) {
	handler := handlers.NewListStoreSubscriptionsHandler(&mockListStoreSubscriptionsService{
		executeFunc: func(context.Context, dto.ListStoreSubscriptionsInput) ([]dto.BillingSubscriptionOutput, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})

	for _, id := range []string{"invalid", uuid.Nil.String()} {
		t.Run(id, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+id+"/billing/subscriptions", nil)
			req.SetPathValue("id", id)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assertBadRequest(t, rec)
		})
	}
}

func TestListStoreSubscriptionsHandler_ServiceError(t *testing.T) {
	handler := handlers.NewListStoreSubscriptionsHandler(&mockListStoreSubscriptionsService{
		executeFunc: func(context.Context, dto.ListStoreSubscriptionsInput) ([]dto.BillingSubscriptionOutput, error) {
			return nil, errors.New("billing unavailable")
		},
	})

	storeID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeID.String()+"/billing/subscriptions", nil)
	req.SetPathValue("id", storeID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}

func TestListStoreSubscriptionsHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewListStoreSubscriptionsHandler(&mockListStoreSubscriptionsService{
		executeFunc: func(context.Context, dto.ListStoreSubscriptionsInput) ([]dto.BillingSubscriptionOutput, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})

	storeID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/"+storeID.String()+"/billing/subscriptions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusMethodNotAllowed, "method not allowed")
}

func TestGetStoreSubscriptionHandler_Success(t *testing.T) {
	storeID := uuid.New()
	subscriptionID := uuid.New()
	planID := uuid.New()
	handler := handlers.NewGetStoreSubscriptionHandler(&mockGetStoreSubscriptionService{
		executeFunc: func(_ context.Context, input dto.GetStoreSubscriptionInput) (dto.BillingSubscriptionOutput, error) {
			if input.StoreID != storeID || input.SubscriptionID != subscriptionID {
				t.Fatalf("unexpected input: %+v", input)
			}
			return dto.BillingSubscriptionOutput{
				ID:       subscriptionID,
				StoreID:  storeID,
				PlanID:   planID,
				Price:    dto.BillingMoneyOutput{AmountMinor: 500000, Currency: "NGN"},
				Duration: 30 * 24 * time.Hour,
				Status:   "active",
			}, nil
		},
	})

	path := "/api/v1/stores/" + storeID.String() + "/billing/subscriptions/" + subscriptionID.String()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.SetPathValue("id", storeID.String())
	req.SetPathValue("subscriptionID", subscriptionID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response struct {
		ID      uuid.UUID `json:"id"`
		StoreID uuid.UUID `json:"store_id"`
		PlanID  uuid.UUID `json:"plan_id"`
		Status  string    `json:"status"`
	}
	if err := decodeTestJSON(rec, &response); err != nil {
		t.Fatal(err)
	}
	if response.ID != subscriptionID ||
		response.StoreID != storeID ||
		response.PlanID != planID ||
		response.Status != "active" {
		t.Fatalf("unexpected subscription response: %+v", response)
	}
}

func TestGetStoreSubscriptionHandler_InvalidIDs(t *testing.T) {
	handler := handlers.NewGetStoreSubscriptionHandler(&mockGetStoreSubscriptionService{
		executeFunc: func(context.Context, dto.GetStoreSubscriptionInput) (dto.BillingSubscriptionOutput, error) {
			t.Fatal("service must not be called")
			return dto.BillingSubscriptionOutput{}, nil
		},
	})

	validID := uuid.NewString()
	tests := []struct {
		name           string
		storeID        string
		subscriptionID string
	}{
		{name: "invalid store ID", storeID: "invalid", subscriptionID: validID},
		{name: "nil store ID", storeID: uuid.Nil.String(), subscriptionID: validID},
		{name: "invalid subscription ID", storeID: validID, subscriptionID: "invalid"},
		{name: "nil subscription ID", storeID: validID, subscriptionID: uuid.Nil.String()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/stores/"+test.storeID+"/billing/subscriptions/"+test.subscriptionID,
				nil,
			)
			req.SetPathValue("id", test.storeID)
			req.SetPathValue("subscriptionID", test.subscriptionID)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assertBadRequest(t, rec)
		})
	}
}

func TestGetStoreSubscriptionHandler_NotFound(t *testing.T) {
	handler := handlers.NewGetStoreSubscriptionHandler(&mockGetStoreSubscriptionService{
		executeFunc: func(context.Context, dto.GetStoreSubscriptionInput) (dto.BillingSubscriptionOutput, error) {
			return dto.BillingSubscriptionOutput{}, application_errors.ErrBillingSubscriptionNotFound
		},
	})

	storeID := uuid.New()
	subscriptionID := uuid.New()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores/"+storeID.String()+"/billing/subscriptions/"+subscriptionID.String(),
		nil,
	)
	req.SetPathValue("id", storeID.String())
	req.SetPathValue("subscriptionID", subscriptionID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "subscription not found")
}

func TestGetStoreSubscriptionHandler_ServiceError(t *testing.T) {
	handler := handlers.NewGetStoreSubscriptionHandler(&mockGetStoreSubscriptionService{
		executeFunc: func(context.Context, dto.GetStoreSubscriptionInput) (dto.BillingSubscriptionOutput, error) {
			return dto.BillingSubscriptionOutput{}, errors.New("billing unavailable")
		},
	})

	storeID := uuid.New()
	subscriptionID := uuid.New()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/stores/"+storeID.String()+"/billing/subscriptions/"+subscriptionID.String(),
		nil,
	)
	req.SetPathValue("id", storeID.String())
	req.SetPathValue("subscriptionID", subscriptionID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}

func TestGetStoreSubscriptionHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewGetStoreSubscriptionHandler(&mockGetStoreSubscriptionService{
		executeFunc: func(context.Context, dto.GetStoreSubscriptionInput) (dto.BillingSubscriptionOutput, error) {
			t.Fatal("service must not be called")
			return dto.BillingSubscriptionOutput{}, nil
		},
	})

	storeID := uuid.New()
	subscriptionID := uuid.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/stores/"+storeID.String()+"/billing/subscriptions/"+subscriptionID.String(),
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusMethodNotAllowed, "method not allowed")
}

var (
	_ ports.ListBillingPlansService       = (*mockListBillingPlansService)(nil)
	_ ports.GetBillingPlanService         = (*mockGetBillingPlanService)(nil)
	_ ports.ListStoreSubscriptionsService = (*mockListStoreSubscriptionsService)(nil)
	_ ports.GetStoreSubscriptionService   = (*mockGetStoreSubscriptionService)(nil)
)
