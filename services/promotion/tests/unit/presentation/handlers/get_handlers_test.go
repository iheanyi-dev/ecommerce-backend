package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	presentationhandlers "github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/handlers"
)

type getPromotionByIDServiceStub struct {
	result dtos.PromotionDTO
	err    error
	input  dtos.GetPromotionByIDInput
	calls  int
}

func (s *getPromotionByIDServiceStub) Execute(
	_ context.Context,
	input dtos.GetPromotionByIDInput,
) (dtos.PromotionDTO, error) {
	s.calls++
	s.input = input
	return s.result, s.err
}

type getPromosForUserServiceStub struct {
	result []dtos.PromotionDTO
	err    error
	input  dtos.GetPromosForUserInput
	calls  int
}

func (s *getPromosForUserServiceStub) Execute(
	_ context.Context,
	input dtos.GetPromosForUserInput,
) ([]dtos.PromotionDTO, error) {
	s.calls++
	s.input = input
	return s.result, s.err
}

type getPromoForUserServiceStub struct {
	result dtos.PromotionDTO
	err    error
	input  dtos.GetPromoForUserInput
	calls  int
}

func (s *getPromoForUserServiceStub) Execute(
	_ context.Context,
	input dtos.GetPromoForUserInput,
) (dtos.PromotionDTO, error) {
	s.calls++
	s.input = input
	return s.result, s.err
}

func authenticatedRequest(method, target string) *http.Request {
	userID := uuid.New()
	request := httptest.NewRequest(method, target, nil)
	ctx := ports.WithAuthenticatedIdentity(request.Context(), ports.AuthenticatedIdentity{
		UserID: userID,
		Role:   "customer",
	})
	return request.WithContext(ctx)
}

func TestGetPromotionByIDHandler(t *testing.T) {
	promotionID := uuid.NewString()
	service := &getPromotionByIDServiceStub{
		result: dtos.PromotionDTO{ID: promotionID},
	}
	handler := presentationhandlers.NewGetPromotionByIDHandlerWithService(service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/promotions/"+promotionID, nil)
	request.SetPathValue("id", promotionID)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body)
	}
	if service.calls != 1 || service.input.PromotionID != promotionID {
		t.Fatalf("service call/input = %d/%q", service.calls, service.input.PromotionID)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["id"] != promotionID {
		t.Fatalf("response ID = %v, want %s", response["id"], promotionID)
	}
}

func TestGetPromotionByIDHandler_MethodNotAllowed(t *testing.T) {
	service := &getPromotionByIDServiceStub{}
	handler := presentationhandlers.NewGetPromotionByIDHandlerWithService(service)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if service.calls != 0 {
		t.Fatalf("service called %d times, want 0", service.calls)
	}
}

func TestGetPromosForUserHandler(t *testing.T) {
	tests := []struct {
		name          string
		target        string
		authenticated bool
		wantStatus    int
		wantLimit     int
		wantOffset    int
	}{
		{
			name:          "defaults pagination",
			target:        "/api/v1/promotions/users/me",
			authenticated: true,
			wantStatus:    http.StatusOK,
			wantLimit:     20,
			wantOffset:    0,
		},
		{
			name:          "accepts pagination",
			target:        "/api/v1/promotions/users/me?limit=10&offset=30",
			authenticated: true,
			wantStatus:    http.StatusOK,
			wantLimit:     10,
			wantOffset:    30,
		},
		{
			name:          "rejects invalid limit",
			target:        "/api/v1/promotions/users/me?limit=101",
			authenticated: true,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:       "requires authentication",
			target:     "/api/v1/promotions/users/me",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &getPromosForUserServiceStub{}
			handler := presentationhandlers.NewGetPromosForUserHandlerWithService(service)

			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			if tt.authenticated {
				userID := uuid.New()
				request = request.WithContext(ports.WithAuthenticatedIdentity(
					request.Context(),
					ports.AuthenticatedIdentity{UserID: userID, Role: "customer"},
				))
				service.result = []dtos.PromotionDTO{}
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body)
			}
			if tt.authenticated && tt.wantStatus == http.StatusOK {
				if service.input.Limit != tt.wantLimit || service.input.Offset != tt.wantOffset {
					t.Fatalf("pagination = %d/%d, want %d/%d",
						service.input.Limit, service.input.Offset, tt.wantLimit, tt.wantOffset)
				}
				if service.input.CustomerID == "" {
					t.Fatal("customer ID was not propagated")
				}
			}
			if !tt.authenticated && service.calls != 0 {
				t.Fatalf("service called %d times without authentication", service.calls)
			}
		})
	}
}

func TestGetPromoForUserHandler(t *testing.T) {
	tests := []struct {
		name          string
		target        string
		authenticated bool
		serviceErr    error
		wantStatus    int
	}{
		{
			name:          "returns best promotion",
			target:        "/api/v1/promotions/best?store_id=" + uuid.NewString() + "&product_id=" + uuid.NewString(),
			authenticated: true,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "maps invalid query",
			target:        "/api/v1/promotions/best",
			authenticated: true,
			serviceErr:    apperrors.ErrInvalidPromotionQuery,
			wantStatus:    http.StatusBadRequest,
		},
		{
			name:       "requires authentication",
			target:     "/api/v1/promotions/best",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &getPromoForUserServiceStub{err: tt.serviceErr}
			handler := presentationhandlers.NewGetPromoForUserHandlerWithService(service)

			var request *http.Request
			if tt.authenticated {
				request = authenticatedRequest(http.MethodGet, tt.target)
			} else {
				request = httptest.NewRequest(http.MethodGet, tt.target, nil)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body)
			}
			if tt.authenticated && service.calls != 1 {
				t.Fatalf("service calls = %d, want 1", service.calls)
			}
			if !tt.authenticated && service.calls != 0 {
				t.Fatalf("service called %d times without authentication", service.calls)
			}
			if tt.name == "returns best promotion" {
				if service.input.StoreID == "" || service.input.ProductID == "" || service.input.CustomerID == "" {
					t.Fatalf("incomplete input passed to service: %+v", service.input)
				}
			}
		})
	}
}

func TestGetPromotionByIDHandler_ServiceError(t *testing.T) {
	service := &getPromotionByIDServiceStub{err: errors.New("database unavailable")}
	handler := presentationhandlers.NewGetPromotionByIDHandlerWithService(service)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetPathValue("id", uuid.NewString())
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
