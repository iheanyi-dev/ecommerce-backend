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

type listOrdersMock struct {
	called         bool
	page, pageSize int
	result         *dto.ListStoreOrdersOutput
	err            error
}

func (m *listOrdersMock) Execute(_ context.Context, page, size int) (*dto.ListStoreOrdersOutput, error) {
	m.called, m.page, m.pageSize = true, page, size
	return m.result, m.err
}

type getOrderMock struct {
	called bool
	id     uuid.UUID
	result *dto.OrderOutput
	err    error
}

func (m *getOrderMock) Execute(_ context.Context, id uuid.UUID) (*dto.OrderOutput, error) {
	m.called, m.id = true, id
	return m.result, m.err
}

type addShippingMock struct {
	called bool
	id     uuid.UUID
	amount int64
	err    error
}

func (m *addShippingMock) Execute(_ context.Context, id uuid.UUID, amount int64) error {
	m.called, m.id, m.amount = true, id, amount
	return m.err
}

type markSentMock struct {
	called bool
	id     uuid.UUID
	err    error
}

func (m *markSentMock) Execute(_ context.Context, id uuid.UUID) error {
	m.called, m.id = true, id
	return m.err
}

type cancelOrderMock struct {
	called bool
	id     uuid.UUID
	err    error
}

func (m *cancelOrderMock) Execute(_ context.Context, id uuid.UUID) error {
	m.called, m.id = true, id
	return m.err
}

func TestListStoreOrdersHandler(t *testing.T) {
	service := &listOrdersMock{result: &dto.ListStoreOrdersOutput{}}
	handler := handlers.NewListStoreOrdersHandler(service)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner/orders", nil))

	if rec.Code != http.StatusOK || !service.called || service.page != 1 || service.pageSize != 20 {
		t.Fatalf("status=%d called=%v page=%d pageSize=%d", rec.Code, service.called, service.page, service.pageSize)
	}
}

func TestListStoreOrdersHandler_InvalidPagination(t *testing.T) {
	service := &listOrdersMock{}
	handler := handlers.NewListStoreOrdersHandler(service)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner/orders?page=abc", nil))

	if rec.Code != http.StatusBadRequest || service.called {
		t.Fatalf("status=%d called=%v", rec.Code, service.called)
	}
}

func TestGetStoreOrderHandler(t *testing.T) {
	id := uuid.New()
	service := &getOrderMock{result: &dto.OrderOutput{ID: id}}
	handler := handlers.NewGetStoreOrderHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner/orders/"+id.String(), nil)
	req.SetPathValue("orderID", id.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !service.called || service.id != id {
		t.Fatalf("status=%d called=%v id=%s", rec.Code, service.called, service.id)
	}
}

func TestGetStoreOrderHandler_InvalidID(t *testing.T) {
	service := &getOrderMock{}
	handler := handlers.NewGetStoreOrderHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stores/owner/orders/invalid", nil)
	req.SetPathValue("orderID", "invalid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest || service.called {
		t.Fatalf("status=%d called=%v", rec.Code, service.called)
	}
}

func TestAddShippingHandler(t *testing.T) {
	id := uuid.New()
	service := &addShippingMock{}
	handler := handlers.NewAddShippingHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+id.String()+"/shipping", strings.NewReader(`{"shipping_amount":1250}`))
	req.SetPathValue("orderID", id.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !service.called || service.id != id || service.amount != 1250 {
		t.Fatalf("status=%d called=%v id=%s amount=%d", rec.Code, service.called, service.id, service.amount)
	}
}

func TestAddShippingHandler_RejectsNegativeAmount(t *testing.T) {
	id := uuid.New()
	service := &addShippingMock{}
	handler := handlers.NewAddShippingHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+id.String()+"/shipping", strings.NewReader(`{"shipping_amount":-1}`))
	req.SetPathValue("orderID", id.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest || service.called {
		t.Fatalf("status=%d called=%v", rec.Code, service.called)
	}
}

func TestMarkOrderSentHandler(t *testing.T) {
	id := uuid.New()
	service := &markSentMock{}
	handler := handlers.NewMarkOrderSentHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+id.String()+"/sent", nil)
	req.SetPathValue("orderID", id.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !service.called || service.id != id {
		t.Fatalf("status=%d called=%v id=%s", rec.Code, service.called, service.id)
	}
}

func TestCancelStoreOrderHandler(t *testing.T) {
	id := uuid.New()
	service := &cancelOrderMock{}
	handler := handlers.NewCancelStoreOrderHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+id.String()+"/cancel", nil)
	req.SetPathValue("orderID", id.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !service.called || service.id != id {
		t.Fatalf("status=%d called=%v id=%s", rec.Code, service.called, service.id)
	}
}

var (
	_ ports.ListStoreOrdersService  = (*listOrdersMock)(nil)
	_ ports.GetStoreOrderService    = (*getOrderMock)(nil)
	_ ports.AddShippingService      = (*addShippingMock)(nil)
	_ ports.MarkOrderSentService    = (*markSentMock)(nil)
	_ ports.CancelStoreOrderService = (*cancelOrderMock)(nil)
)
