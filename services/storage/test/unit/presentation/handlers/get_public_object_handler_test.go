package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/presentation/handlers"
)

func TestGetPublicObjectHandler(t *testing.T) {
	service := &getPublicObjectServiceMock{
		output: dto.GetPublicObjectOutput{
			Content:     io.NopCloser(io.NewSectionReader(nil, 0, 0)),
			ContentType: "image/png",
		},
	}

	// Replace the empty reader with actual streamed content.
	service.output.Content = io.NopCloser(
		io.Reader(
			&readerWithContent{content: "image bytes"},
		),
	)

	handler := handlers.NewGetPublicObjectHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/storage/public/stores/store-123.png",
		nil,
	)
	req.SetPathValue("reference", "stores/store-123.png")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if got := recorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf(
			"Content-Type = %q, want %q",
			got,
			"image/png",
		)
	}

	if got := recorder.Body.String(); got != "image bytes" {
		t.Fatalf("body = %q, want %q", got, "image bytes")
	}

	if service.input.Reference != "stores/store-123.png" {
		t.Fatalf(
			"reference = %q, want %q",
			service.input.Reference,
			"stores/store-123.png",
		)
	}
}

func TestGetPublicObjectHandler_RejectsNonGET(t *testing.T) {
	handler := handlers.NewGetPublicObjectHandler(
		&getPublicObjectServiceMock{},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/storage/public/stores/store-123.png",
		nil,
	)
	req.SetPathValue("reference", "stores/store-123.png")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusMethodNotAllowed,
		)
	}
}

type getPublicObjectServiceMock struct {
	input  dto.GetPublicObjectInput
	output dto.GetPublicObjectOutput
	err    error
}

func (m *getPublicObjectServiceMock) Execute(
	_ context.Context,
	input dto.GetPublicObjectInput,
) (dto.GetPublicObjectOutput, error) {
	m.input = input

	if m.err != nil {
		return dto.GetPublicObjectOutput{}, m.err
	}

	return m.output, nil
}

type readerWithContent struct {
	content string
	offset  int
}

func (r *readerWithContent) Read(p []byte) (int, error) {
	if r.offset >= len(r.content) {
		return 0, io.EOF
	}

	n := copy(p, r.content[r.offset:])
	r.offset += n

	return n, nil
}
