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

type mockRemoveStaleKnowledgeService struct {
	executeFunc func(context.Context, dto.RemoveStaleKnowledgeInput) (dto.RemoveStaleKnowledgeOutput, error)
}

func (m *mockRemoveStaleKnowledgeService) Execute(ctx context.Context, input dto.RemoveStaleKnowledgeInput) (dto.RemoveStaleKnowledgeOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestRemoveStaleKnowledgeHandler_Success(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewRemoveStaleKnowledgeHandler(&mockRemoveStaleKnowledgeService{
		executeFunc: func(ctx context.Context, input dto.RemoveStaleKnowledgeInput) (dto.RemoveStaleKnowledgeOutput, error) {
			if input.StoreID != storeID || input.Filename != "old.pdf" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return dto.RemoveStaleKnowledgeOutput{
				StoreID:  storeID,
				Filename: input.Filename,
			}, nil
		},
	})

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/stores/"+storeID.String()+"/knowledge",
		strings.NewReader(`{"filename":"old.pdf"}`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRemoveStaleKnowledgeHandler_InvalidJSON(t *testing.T) {
	handler := handlers.NewRemoveStaleKnowledgeHandler(&mockRemoveStaleKnowledgeService{
		executeFunc: func(context.Context, dto.RemoveStaleKnowledgeInput) (dto.RemoveStaleKnowledgeOutput, error) {
			t.Fatal("service must not be called")
			return dto.RemoveStaleKnowledgeOutput{}, nil
		},
	})

	storeID := uuid.New()
	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/stores/"+storeID.String()+"/knowledge",
		strings.NewReader(`{`),
	)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertBadRequest(t, rec)
}

var _ ports.RemoveStaleKnowledgeService = (*mockRemoveStaleKnowledgeService)(nil)
