package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockListStoresService struct {
	executeFunc func(context.Context, string, int, int) (*dto.ListStoresOutput, error)
}

func (m *mockListStoresService) Execute(ctx context.Context, query string, page, pageSize int) (*dto.ListStoresOutput, error) {
	return m.executeFunc(ctx, query, page, pageSize)
}

func TestListStoresHandler_Success(t *testing.T) {
	handler := handlers.NewListStoresHandler(&mockListStoresService{
		executeFunc: func(ctx context.Context, query string, page, pageSize int) (*dto.ListStoresOutput, error) {
			if query != "coffee" || page != 2 || pageSize != 10 {
				t.Fatalf("unexpected input: query=%q page=%d pageSize=%d", query, page, pageSize)
			}

			return &dto.ListStoresOutput{
				Stores:     []dto.StoreOutput{},
				Page:       2,
				PageSize:   10,
				Total:      0,
				TotalPages: 0,
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores?query=coffee&page=2&page_size=10", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestListStoresHandler_DefaultPagination(t *testing.T) {
	handler := handlers.NewListStoresHandler(&mockListStoresService{
		executeFunc: func(ctx context.Context, query string, page, pageSize int) (*dto.ListStoresOutput, error) {
			if page != 1 || pageSize != 20 {
				t.Fatalf("expected defaults 1/20, got %d/%d", page, pageSize)
			}
			return &dto.ListStoresOutput{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestListStoresHandler_InvalidPage(t *testing.T) {
	handler := handlers.NewListStoresHandler(&mockListStoresService{
		executeFunc: func(context.Context, string, int, int) (*dto.ListStoresOutput, error) {
			t.Fatal("service must not be called")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores?page=abc", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

var _ ports.ListStoresService = (*mockListStoresService)(nil)
