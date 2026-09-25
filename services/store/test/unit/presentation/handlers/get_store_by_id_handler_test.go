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

type mockGetStoreByIDService struct {
	executeFunc func(context.Context, uuid.UUID) (dto.StoreOutput, error)
}

func (m *mockGetStoreByIDService) Execute(ctx context.Context, id uuid.UUID) (dto.StoreOutput, error) {
	return m.executeFunc(ctx, id)
}

func TestGetStoreByIDHandler_Success(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewGetStoreByIDHandler(&mockGetStoreByIDService{
		executeFunc: func(ctx context.Context, id uuid.UUID) (dto.StoreOutput, error) {
			if id != storeID {
				t.Fatalf("expected store ID %q, got %q", storeID, id)
			}
			return newStoreOutput(storeID), nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/"+storeID.String(), nil)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetStoreByIDHandler_InvalidID(t *testing.T) {
	handler := handlers.NewGetStoreByIDHandler(&mockGetStoreByIDService{
		executeFunc: func(context.Context, uuid.UUID) (dto.StoreOutput, error) {
			t.Fatal("service must not be called")
			return dto.StoreOutput{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/invalid", nil)
	req.SetPathValue("id", "invalid")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

func TestGetStoreByIDHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewGetStoreByIDHandler(&mockGetStoreByIDService{
		executeFunc: func(context.Context, uuid.UUID) (dto.StoreOutput, error) {
			t.Fatal("service must not be called")
			return dto.StoreOutput{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

var _ ports.GetStoreByIDService = (*mockGetStoreByIDService)(nil)
