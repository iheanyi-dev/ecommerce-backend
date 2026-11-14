package presentation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/dtos"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/presentation/middleware"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/shared/config"
)

type promotionByIDStub struct{}

func (promotionByIDStub) Execute(context.Context, dtos.GetPromotionByIDInput) (dtos.PromotionDTO, error) {
	return dtos.PromotionDTO{}, nil
}

type promosForUserStub struct{}

func (promosForUserStub) Execute(context.Context, dtos.GetPromosForUserInput) ([]dtos.PromotionDTO, error) {
	return []dtos.PromotionDTO{}, nil
}

type promoForUserStub struct{}

func (promoForUserStub) Execute(context.Context, dtos.GetPromoForUserInput) (dtos.PromotionDTO, error) {
	return dtos.PromotionDTO{}, nil
}

func newTestRouter() http.Handler {
	return presentation.NewRouter(
		handlers.NewGetPromotionByIDHandlerWithService(promotionByIDStub{}),
		handlers.NewGetPromosForUserHandlerWithService(promosForUserStub{}),
		handlers.NewGetPromoForUserHandlerWithService(promoForUserStub{}),
		&config.Config{
			ServiceAuthName:   "test-service",
			ServiceAuthSecret: "test-secret",
		},
		nil,
	)
}

func TestRouterServiceAuthenticatedPromotionLookup(t *testing.T) {
	router := newTestRouter()

	tests := []struct {
		name       string
		service    string
		secret     string
		wantStatus int
	}{
		{name: "missing credentials", wantStatus: http.StatusUnauthorized},
		{name: "wrong service", service: "other", secret: "test-secret", wantStatus: http.StatusUnauthorized},
		{name: "wrong secret", service: "test-service", secret: "wrong", wantStatus: http.StatusUnauthorized},
		{name: "valid credentials", service: "test-service", secret: "test-secret", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/promotions/"+uuid.NewString(), nil)
			req.Header.Set(middleware.ServiceNameHeader, tt.service)
			req.Header.Set(middleware.ServiceSecretHeader, tt.secret)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if rec.Header().Get(middleware.RequestIDHeader) == "" {
				t.Fatal("expected request ID response header")
			}
		})
	}
}

func TestRouterUserEndpointsRequireIdentityInsteadOfServiceCredentials(t *testing.T) {
	router := newTestRouter()

	tests := []struct {
		name string
		path string
	}{
		{name: "user promotions", path: "/api/v1/promotions/users/me"},
		{name: "best promotion", path: "/api/v1/promotions/best?store_id=" + uuid.NewString() + "&product_id=" + uuid.NewString()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
		})
	}
}

func TestRouterUserEndpointsAcceptAuthenticatedIdentity(t *testing.T) {
	router := newTestRouter()
	userID := uuid.New()

	tests := []struct {
		name string
		path string
	}{
		{name: "user promotions", path: "/api/v1/promotions/users/me"},
		{name: "best promotion", path: "/api/v1/promotions/best?store_id=" + uuid.NewString() + "&product_id=" + uuid.NewString()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(middleware.AuthenticatedUserIDHeader, userID.String())
			req.Header.Set(middleware.AuthenticatedRoleHeader, "customer")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("authenticated status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
			}
		})
	}
}

func TestRouterRejectsPartialIdentity(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/promotions/users/me", nil)
	req.Header.Set(middleware.AuthenticatedUserIDHeader, uuid.NewString())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRouterDoesNotRequireServiceCredentialsForUserEndpoint(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/promotions/users/me", nil)
	req.Header.Set(middleware.AuthenticatedUserIDHeader, uuid.NewString())
	req.Header.Set(middleware.AuthenticatedRoleHeader, "customer")
	// Deliberately omit service authentication headers.
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; user routes must not require service credentials", rec.Code, http.StatusOK)
	}
}

func TestRouterIdentityContextIsPropagated(t *testing.T) {
	userID := uuid.New()
	ctx := ports.WithAuthenticatedIdentity(
		context.Background(),
		ports.AuthenticatedIdentity{UserID: userID, Role: "customer"},
	)

	if identity, ok := ports.AuthenticatedIdentityFromContext(ctx); !ok || identity.UserID != userID {
		t.Fatal("expected authenticated identity in context")
	}
}
