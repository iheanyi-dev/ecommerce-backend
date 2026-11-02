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

type mockChangeStoreStatusService struct {
	executeFunc func(context.Context, dto.ChangeStoreStatusInput) (dto.StoreOutput, error)
}

func (m *mockChangeStoreStatusService) Execute(ctx context.Context, input dto.ChangeStoreStatusInput) (dto.StoreOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestChangeStoreStatusHandler_Success(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewChangeStoreStatusHandler(&mockChangeStoreStatusService{
		executeFunc: func(ctx context.Context, input dto.ChangeStoreStatusInput) (dto.StoreOutput, error) {
			if input.StoreID != storeID || input.Status != string(valueobjects.StatusInactive) {
				t.Fatalf("unexpected input: %+v", input)
			}
			return newStoreOutput(storeID), nil
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/"+storeID.String()+"/status", strings.NewReader(`{"status":"inactive"}`))
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestChangeStoreStatusHandler_InvalidJSON(t *testing.T) {
	handler := handlers.NewChangeStoreStatusHandler(&mockChangeStoreStatusService{
		executeFunc: func(context.Context, dto.ChangeStoreStatusInput) (dto.StoreOutput, error) {
			t.Fatal("service must not be called")
			return dto.StoreOutput{}, nil
		},
	})

	storeID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/"+storeID.String()+"/status", strings.NewReader(`{`))
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

var _ ports.ChangeStoreStatusService = (*mockChangeStoreStatusService)(nil)
