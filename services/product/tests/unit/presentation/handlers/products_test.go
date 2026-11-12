package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/presentation/handlers"
)

type createUseCaseFake struct {
	called bool
	input  use_cases.CreateProductInput
	result *entities.Product
	err    error
}

func (f *createUseCaseFake) Execute(_ context.Context, input use_cases.CreateProductInput) (*entities.Product, error) {
	f.called = true
	f.input = input
	return f.result, f.err
}

type getUseCaseFake struct {
	called bool
	id     string
	result use_cases.ProductResult
	err    error
}

func (f *getUseCaseFake) Execute(_ context.Context, id string) (use_cases.ProductResult, error) {
	f.called = true
	f.id = id
	return f.result, f.err
}

type updateUseCaseFake struct{}

func (f *updateUseCaseFake) Execute(context.Context, use_cases.UpdateProductInput) (*entities.Product, error) {
	return nil, errors.New("not configured")
}

type deleteUseCaseFake struct{}

func (f *deleteUseCaseFake) Execute(context.Context, use_cases.DeleteProductInput) error {
	return errors.New("not configured")
}

func newHandler(create *createUseCaseFake, get *getUseCaseFake) *handlers.ProductHandler {
	return handlers.NewProductHandler(handlers.ProductUseCases{
		Create: create,
		Get:    get,
		Update: &updateUseCaseFake{},
		Delete: &deleteUseCaseFake{},
	})
}

func authenticatedRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	identity := ports.AuthenticatedIdentity{
		UserID: uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		Role:   "vendor",
	}
	return request.WithContext(ports.WithAuthenticatedIdentity(request.Context(), identity))
}

func TestCreateRequiresAuthenticatedIdentity(t *testing.T) {
	create := &createUseCaseFake{}
	handler := newHandler(create, &getUseCaseFake{})
	response := httptest.NewRecorder()

	handler.Create(response, httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBufferString(`{}`)))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if create.called {
		t.Fatal("create use case called without authenticated identity")
	}
}

func TestCreateRejectsMalformedJSON(t *testing.T) {
	create := &createUseCaseFake{}
	handler := newHandler(create, &getUseCaseFake{})
	response := httptest.NewRecorder()

	handler.Create(response, authenticatedRequest(http.MethodPost, "/api/v1/products", `{"name":`))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if create.called {
		t.Fatal("create use case called for malformed JSON")
	}
}

func TestCreateRejectsInvalidMoney(t *testing.T) {
	create := &createUseCaseFake{}
	handler := newHandler(create, &getUseCaseFake{})
	response := httptest.NewRecorder()

	handler.Create(response, authenticatedRequest(
		http.MethodPost,
		"/api/v1/products",
		`{"name":"Keyboard","description":"Mechanical","price":1000,"currency":"BAD","quantity":2}`,
	))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if create.called {
		t.Fatal("create use case called for invalid currency")
	}
}

func TestCreatePassesTrustedOwnerAndReturnsCreatedProduct(t *testing.T) {
	price, err := valueobjects.NewMoney(250000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	product, err := entities.NewProduct(
		"22222222-2222-4222-8222-222222222222",
		"Keyboard",
		"Mechanical keyboard",
		price,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}

	create := &createUseCaseFake{result: product}
	handler := newHandler(create, &getUseCaseFake{})
	response := httptest.NewRecorder()

	handler.Create(response, authenticatedRequest(
		http.MethodPost,
		"/api/v1/products",
		`{"name":"Keyboard","description":"Mechanical keyboard","price":250000,"currency":"NGN","quantity":5}`,
	))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	if !create.called {
		t.Fatal("create use case was not called")
	}
	if create.input.OwnerID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("owner ID = %q; expected trusted identity", create.input.OwnerID)
	}
	if create.input.Name != "Keyboard" || create.input.Quantity != 5 {
		t.Fatalf("unexpected create input: %+v", create.input)
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != product.ID() {
		t.Fatalf("response id = %v, want %q", body["id"], product.ID())
	}
	if body["currency"] != "NGN" {
		t.Fatalf("response currency = %v, want NGN", body["currency"])
	}
}

func TestGetPassesProductID(t *testing.T) {
	price, err := valueobjects.NewMoney(250000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	product, err := entities.NewProduct(
		"22222222-2222-4222-8222-222222222222",
		"Keyboard",
		"Mechanical keyboard",
		price,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}

	get := &getUseCaseFake{
		result: use_cases.ProductResult{
			Product: product,
		},
	}
	handler := newHandler(&createUseCaseFake{}, get)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/products/product-123", nil)
	request.SetPathValue("id", "product-123")

	handler.Get(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if !get.called {
		t.Fatal("get use case was not called")
	}
	if get.id != "product-123" {
		t.Fatalf("product ID = %q, want product-123", get.id)
	}
}

type changeVideoHandlerFake struct {
	called  bool
	input   use_cases.ChangeProductVideoInput
	result  *entities.Product
	content []byte
	err     error
}

func (f *changeVideoHandlerFake) Execute(_ context.Context, input use_cases.ChangeProductVideoInput) (*entities.Product, error) {
	f.called = true
	f.input = input
	if input.Content != nil {
		content, err := io.ReadAll(input.Content)
		if err != nil {
			return nil, err
		}
		f.content = content
	}
	return f.result, f.err
}

type removeVideoHandlerFake struct {
	called bool
	input  use_cases.RemoveProductVideoInput
	err    error
}

func (f *removeVideoHandlerFake) Execute(_ context.Context, input use_cases.RemoveProductVideoInput) error {
	f.called = true
	f.input = input
	return f.err
}

func videoRequest(t *testing.T, authenticated bool, contentType string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="video"; filename="clip"`)
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

	request := httptest.NewRequest(http.MethodPut, "/api/v1/products/product-123/video", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.SetPathValue("id", "product-123")

	if authenticated {
		identity := ports.AuthenticatedIdentity{
			UserID: uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			Role:   "vendor",
		}
		request = request.WithContext(ports.WithAuthenticatedIdentity(request.Context(), identity))
	}
	return request
}

func videoTestProduct(t *testing.T) *entities.Product {
	t.Helper()
	price, err := valueobjects.NewMoney(250000, "NGN")
	if err != nil {
		t.Fatal(err)
	}
	product, err := entities.NewProduct(
		"22222222-2222-4222-8222-222222222222",
		"Keyboard", "Mechanical keyboard", price, 5,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := product.SetVideo("products/product-123/clip.mp4"); err != nil {
		t.Fatal(err)
	}
	return product
}

func TestChangeVideoUploadsMP4(t *testing.T) {
	content := []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}
	change := &changeVideoHandlerFake{result: videoTestProduct(t)}
	handler := handlers.NewProductHandler(handlers.ProductUseCases{ChangeVideo: change})
	response := httptest.NewRecorder()

	handler.ChangeVideo(response, videoRequest(t, true, "video/mp4", content))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	if !change.called {
		t.Fatal("change video use case was not called")
	}
	if change.input.OwnerID != "11111111-1111-4111-8111-111111111111" ||
		change.input.ProductID != "product-123" ||
		change.input.ContentType != "video/mp4" {
		t.Fatalf("unexpected input: %+v", change.input)
	}
	if !bytes.Equal(change.content, content) {
		t.Fatalf("uploaded content = %v, want %v", change.content, content)
	}
}

func TestChangeVideoRejectsInvalidSignature(t *testing.T) {
	change := &changeVideoHandlerFake{result: videoTestProduct(t)}
	handler := handlers.NewProductHandler(handlers.ProductUseCases{ChangeVideo: change})
	response := httptest.NewRecorder()

	handler.ChangeVideo(response, videoRequest(t, true, "video/mp4", []byte("not a video file")))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if change.called {
		t.Fatal("use case called for invalid video")
	}
}

func TestChangeVideoRequiresAuthentication(t *testing.T) {
	change := &changeVideoHandlerFake{result: videoTestProduct(t)}
	handler := handlers.NewProductHandler(handlers.ProductUseCases{ChangeVideo: change})
	response := httptest.NewRecorder()
	content := []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}

	handler.ChangeVideo(response, videoRequest(t, false, "video/mp4", content))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if change.called {
		t.Fatal("use case called without authentication")
	}
}

func TestRemoveVideoSucceeds(t *testing.T) {
	remove := &removeVideoHandlerFake{}
	handler := handlers.NewProductHandler(handlers.ProductUseCases{RemoveVideo: remove})
	request := authenticatedRequest(http.MethodDelete, "/api/v1/products/product-123/video", "")
	request.SetPathValue("id", "product-123")
	response := httptest.NewRecorder()

	handler.RemoveVideo(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if !remove.called {
		t.Fatal("remove video use case was not called")
	}
	if remove.input.OwnerID != "11111111-1111-4111-8111-111111111111" ||
		remove.input.ProductID != "product-123" {
		t.Fatalf("unexpected input: %+v", remove.input)
	}
}

func TestRemoveVideoRequiresAuthentication(t *testing.T) {
	remove := &removeVideoHandlerFake{}
	handler := handlers.NewProductHandler(handlers.ProductUseCases{RemoveVideo: remove})
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/products/product-123/video", nil)
	request.SetPathValue("id", "product-123")
	response := httptest.NewRecorder()

	handler.RemoveVideo(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if remove.called {
		t.Fatal("use case called without authentication")
	}
}
