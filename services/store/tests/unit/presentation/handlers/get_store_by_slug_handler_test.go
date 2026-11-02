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

type mockGetStoreBySlugService struct {
	executeFunc func(context.Context, string) (*dto.StoreOutput, error)
}

func (m *mockGetStoreBySlugService) Execute(ctx context.Context, slug string) (*dto.StoreOutput, error) {
	return m.executeFunc(ctx, slug)
}

func TestGetStoreBySlugHandler_Success(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewGetStoreBySlugHandler(&mockGetStoreBySlugService{
		executeFunc: func(ctx context.Context, slug string) (*dto.StoreOutput, error) {
			if slug != "my-store" {
				t.Fatalf("expected slug %q, got %q", "my-store", slug)
			}
			result := newStoreOutput(storeID)
			return &result, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/my-store", nil)
	req.SetPathValue("slug", "my-store")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetStoreBySlugHandler_EmptySlug(t *testing.T) {
	handler := handlers.NewGetStoreBySlugHandler(&mockGetStoreBySlugService{
		executeFunc: func(context.Context, string) (*dto.StoreOutput, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/", nil)
	req.SetPathValue("slug", "")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

func TestGetStoreBySlugHandler_MethodNotAllowed(t *testing.T) {
	handler := handlers.NewGetStoreBySlugHandler(&mockGetStoreBySlugService{
		executeFunc: func(context.Context, string) (*dto.StoreOutput, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stores/my-store", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

var _ ports.GetStoreBySlugService = (*mockGetStoreBySlugService)(nil)
