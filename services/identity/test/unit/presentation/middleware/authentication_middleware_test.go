package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

type mockTokenService struct {
	validateFunc func(
		ctx context.Context,
		token string,
	) (ports.AuthenticatedIdentity, error)
}

func (m *mockTokenService) GenerateAccessToken(
	ctx context.Context,
	userID user.UserID,
	role user.Role,
) (string, error) {
	return "test-token", nil
}

func (m *mockTokenService) ValidateAccessToken(
	ctx context.Context,
	token string,
) (ports.AuthenticatedIdentity, error) {
	return m.validateFunc(ctx, token)
}

func TestAuthenticationMiddleware_RequiresAuthorizationHeader(
	t *testing.T,
) {
	tokenService := &mockTokenService{
		validateFunc: func(
			ctx context.Context,
			token string,
		) (ports.AuthenticatedIdentity, error) {
			t.Fatal("token validation should not be called")
			return ports.AuthenticatedIdentity{}, nil
		},
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("protected handler should not be called")
	})

	handler := authMiddleware.RequireAuthentication(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}

func TestAuthenticationMiddleware_RejectsInvalidAuthorizationHeader(
	t *testing.T,
) {
	tokenService := &mockTokenService{
		validateFunc: func(
			ctx context.Context,
			token string,
		) (ports.AuthenticatedIdentity, error) {
			t.Fatal("token validation should not be called")
			return ports.AuthenticatedIdentity{}, nil
		},
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("protected handler should not be called")
	})

	handler := authMiddleware.RequireAuthentication(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Basic abc123",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}

func TestAuthenticationMiddleware_RejectsInvalidToken(
	t *testing.T,
) {
	tokenService := &mockTokenService{
		validateFunc: func(
			ctx context.Context,
			token string,
		) (ports.AuthenticatedIdentity, error) {
			if token != "invalid-token" {
				t.Fatalf(
					"expected invalid-token, got %q",
					token,
				)
			}

			return ports.AuthenticatedIdentity{}, errors.New(
				"invalid token",
			)
		},
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("protected handler should not be called")
	})

	handler := authMiddleware.RequireAuthentication(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer invalid-token",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}

func TestAuthenticationMiddleware_AllowsValidToken(
	t *testing.T,
) {
	expectedIdentity := ports.AuthenticatedIdentity{
		UserID: "test-user-id",
		Role:   "vendor",
	}

	tokenService := &mockTokenService{
		validateFunc: func(
			ctx context.Context,
			token string,
		) (ports.AuthenticatedIdentity, error) {
			if token != "valid-token" {
				t.Fatalf(
					"expected valid-token, got %q",
					token,
				)
			}

			return expectedIdentity, nil
		},
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		identity, ok := middleware.AuthenticatedIdentity(
			r.Context(),
		)

		if !ok {
			t.Fatal(
				"expected authenticated identity in request context",
			)
		}

		if identity.UserID != expectedIdentity.UserID {
			t.Fatalf(
				"expected user ID %q, got %q",
				expectedIdentity.UserID,
				identity.UserID,
			)
		}

		if identity.Role != expectedIdentity.Role {
			t.Fatalf(
				"expected role %q, got %q",
				expectedIdentity.Role,
				identity.Role,
			)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddleware.RequireAuthentication(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer valid-token",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			recorder.Code,
		)
	}
}

func TestAuthenticatedIdentity_ReturnsFalseWhenMissing(
	t *testing.T,
) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	_, ok := middleware.AuthenticatedIdentity(
		request.Context(),
	)

	if ok {
		t.Fatal(
			"expected authenticated identity to be missing",
		)
	}
}

func TestAuthenticationMiddleware_RejectsBearerWithoutToken(
	t *testing.T,
) {
	tokenService := &mockTokenService{
		validateFunc: func(
			ctx context.Context,
			token string,
		) (ports.AuthenticatedIdentity, error) {
			t.Fatal("token validation should not be called")
			return ports.AuthenticatedIdentity{}, nil
		},
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("protected handler should not be called")
	})

	handler := authMiddleware.RequireAuthentication(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}

func TestAuthenticationMiddleware_RejectsAuthorizationHeaderWithExtraFields(
	t *testing.T,
) {
	tokenService := &mockTokenService{
		validateFunc: func(
			ctx context.Context,
			token string,
		) (ports.AuthenticatedIdentity, error) {
			t.Fatal("token validation should not be called")
			return ports.AuthenticatedIdentity{}, nil
		},
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("protected handler should not be called")
	})

	handler := authMiddleware.RequireAuthentication(next)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer valid-token extra-value",
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}