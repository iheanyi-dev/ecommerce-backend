package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockChangeStorePlanService struct {
	executeFunc func(context.Context, dto.ChangeStorePlanInput) (dto.StoreOutput, error)
}

func (m *mockChangeStorePlanService) Execute(ctx context.Context, input dto.ChangeStorePlanInput) (dto.StoreOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestChangeStorePlanHandler_Success(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewChangeStorePlanHandler(&mockChangeStorePlanService{
		executeFunc: func(ctx context.Context, input dto.ChangeStorePlanInput) (dto.StoreOutput, error) {
			if input.StoreID != storeID || input.PlanType != string(valueobjects.PlanTypePremium) {
				t.Fatalf("unexpected input: %+v", input)
			}
			return newStoreOutput(storeID), nil
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/"+storeID.String()+"/plan", strings.NewReader(`{"plan_type":"premium"}`))
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestChangeStorePlanHandler_InvalidJSON(t *testing.T) {
	handler := handlers.NewChangeStorePlanHandler(&mockChangeStorePlanService{
		executeFunc: func(context.Context, dto.ChangeStorePlanInput) (dto.StoreOutput, error) {
			t.Fatal("service must not be called")
			return dto.StoreOutput{}, nil
		},
	})

	storeID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/"+storeID.String()+"/plan", strings.NewReader(`{`))
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

var _ ports.ChangeStorePlanService = (*mockChangeStorePlanService)(nil)
