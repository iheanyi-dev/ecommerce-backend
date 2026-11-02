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
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockUpdateStoreService struct {
	executeFunc func(context.Context, dto.UpdateStoreInput) (dto.UpdateStoreOutput, error)
}

func (m *mockUpdateStoreService) Execute(ctx context.Context, input dto.UpdateStoreInput) (dto.UpdateStoreOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestUpdateStoreHandler_Success(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewUpdateStoreHandler(&mockUpdateStoreService{
		executeFunc: func(ctx context.Context, input dto.UpdateStoreInput) (dto.UpdateStoreOutput, error) {
			if input.StoreID != storeID ||
				input.Name != "Updated Store" ||
				input.Slug != "updated-store" ||
				input.Description != "Updated description" {
				t.Fatalf("unexpected input: %+v", input)
			}

			return newStoreOutput(storeID), nil
		},
	})

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/stores/"+storeID.String(),
		strings.NewReader(`{"name":"Updated Store","slug":"updated-store","description":"Updated description"}`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUpdateStoreHandler_InvalidJSON(t *testing.T) {
	handler := handlers.NewUpdateStoreHandler(&mockUpdateStoreService{
		executeFunc: func(context.Context, dto.UpdateStoreInput) (dto.UpdateStoreOutput, error) {
			t.Fatal("service must not be called")
			return dto.UpdateStoreOutput{}, nil
		},
	})

	storeID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/"+storeID.String(), strings.NewReader(`{"name":`))
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

var _ ports.UpdateStoreService = (*mockUpdateStoreService)(nil)
