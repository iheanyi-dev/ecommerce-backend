package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockGetStoreByOwnerService struct {
	executeFunc func(context.Context) (*dto.StoreOutput, error)
}

func (m *mockGetStoreByOwnerService) Execute(ctx context.Context) (*dto.StoreOutput, error) {
	return m.executeFunc(ctx)
}

func TestGetStoreByOwnerHandler_Success(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewGetStoreByOwnerHandler(&mockGetStoreByOwnerService{
		executeFunc: func(ctx context.Context) (*dto.StoreOutput, error) {
			result := newStoreOutput(storeID)
			return &result, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetStoreByOwnerHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewGetStoreByOwnerHandler(&mockGetStoreByOwnerService{
		executeFunc: func(context.Context) (*dto.StoreOutput, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/owner", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

var _ ports.GetStoreByOwnerService = (*mockGetStoreByOwnerService)(nil)
