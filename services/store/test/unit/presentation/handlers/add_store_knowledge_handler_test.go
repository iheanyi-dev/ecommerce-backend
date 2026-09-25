package handlers_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/handlers"
)

type mockAddStoreKnowledgeService struct {
	executeFunc func(context.Context, dto.AddStoreKnowledgeInput) (dto.AddStoreKnowledgeOutput, error)
}

func (m *mockAddStoreKnowledgeService) Execute(ctx context.Context, input dto.AddStoreKnowledgeInput) (dto.AddStoreKnowledgeOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestAddStoreKnowledgeHandler_Success(t *testing.T) {
	storeID := uuid.New()
	content := []byte("knowledge")

	handler := handlers.NewAddStoreKnowledgeHandler(&mockAddStoreKnowledgeService{
		executeFunc: func(ctx context.Context, input dto.AddStoreKnowledgeInput) (dto.AddStoreKnowledgeOutput, error) {
			if input.StoreID != storeID ||
				!bytes.Equal(input.File.Content, content) ||
				input.File.Filename != "knowledge.pdf" ||
				input.File.ContentType != "application/pdf" {
				t.Fatalf("unexpected input")
			}
			return dto.AddStoreKnowledgeOutput{StoreID: storeID}, nil
		},
	})

	req := newKnowledgeMultipartRequest(t, content)
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func newKnowledgeMultipartRequest(t *testing.T, content []byte) *http.Request {
	t.Helper()

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="knowledge.pdf"`)
	header.Set("Content-Type", "application/pdf")

	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/knowledge", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

var _ ports.AddStoreKnowledgeService = (*mockAddStoreKnowledgeService)(nil)
