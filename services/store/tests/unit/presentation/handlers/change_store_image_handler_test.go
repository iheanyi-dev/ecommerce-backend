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

type mockChangeStoreImageService struct {
	executeFunc func(context.Context, dto.ChangeStoreImageInput) (dto.ChangeStoreImageOutput, error)
}

func (m *mockChangeStoreImageService) Execute(ctx context.Context, input dto.ChangeStoreImageInput) (dto.ChangeStoreImageOutput, error) {
	return m.executeFunc(ctx, input)
}

func TestChangeStoreImageHandler_WithImage(t *testing.T) {
	storeID := uuid.New()
	content := []byte("image")

	handler := handlers.NewChangeStoreImageHandler(&mockChangeStoreImageService{
		executeFunc: func(ctx context.Context, input dto.ChangeStoreImageInput) (dto.ChangeStoreImageOutput, error) {
			if input.StoreID != storeID || input.Image == nil {
				t.Fatalf("unexpected input")
			}
			if !bytes.Equal(input.Image.Content, content) ||
				input.Image.Filename != "store.webp" ||
				input.Image.ContentType != "image/webp" {
				t.Fatalf("unexpected image input")
			}
			return newStoreOutput(storeID), nil
		},
	})

	req := newMultipartRequest(t, http.MethodPatch, "image", "store.webp", "image/webp", content)
	req.URL.Path = "/api/v1/stores/" + storeID.String() + "/image"
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestChangeStoreImageHandler_RemoveImage(t *testing.T) {
	storeID := uuid.New()

	handler := handlers.NewChangeStoreImageHandler(&mockChangeStoreImageService{
		executeFunc: func(ctx context.Context, input dto.ChangeStoreImageInput) (dto.ChangeStoreImageOutput, error) {
			if input.Image != nil {
				t.Fatal("expected nil image")
			}
			return newStoreOutput(storeID), nil
		},
	})

	var body strings.Builder
	writer := multipart.NewWriter(&body)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/stores/"+storeID.String()+"/image", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", storeID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func newMultipartRequest(t *testing.T, method, field, filename, contentType string, content []byte) *http.Request {
	t.Helper()

	var body strings.Builder
	writer := multipart.NewWriter(&body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)

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

	req := httptest.NewRequest(method, "/", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

var _ ports.ChangeStoreImageService = (*mockChangeStoreImageService)(nil)
