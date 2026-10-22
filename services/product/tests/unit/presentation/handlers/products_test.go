package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	result *entities.Product
	err    error
}

func (f *getUseCaseFake) Execute(_ context.Context, id string) (*entities.Product, error) {
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

	get := &getUseCaseFake{result: product}
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
