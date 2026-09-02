package presentation_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

// TestMeEndpoint_Authenticated verifies the complete protected endpoint
// flow.
//
// A valid Bearer token is supplied to the authentication middleware.
// The middleware validates the token and places the authenticated identity
// into the request context. The MeHandler then returns that identity.
func TestMeEndpoint_Authenticated(t *testing.T) {
	tokenService := &fakeTokenService{
		identity: ports.AuthenticatedIdentity{
			UserID: "authenticated-user-id",
			Role:   "vendor",
		},
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	meHandler := handlers.NewMeHandler()

	protectedHandler := authMiddleware.RequireAuthentication(
		meHandler,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer valid-access-token",
	)

	response := httptest.NewRecorder()

	protectedHandler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}

	body := response.Body.String()

	if !strings.Contains(
		body,
		`"user_id":"authenticated-user-id"`,
	) {
		t.Fatalf(
			"expected authenticated user ID in response, got %q",
			body,
		)
	}

	if !strings.Contains(
		body,
		`"role":"vendor"`,
	) {
		t.Fatalf(
			"expected authenticated role in response, got %q",
			body,
		)
	}
}

// TestMeEndpoint_MissingToken verifies that the protected endpoint rejects
// requests that do not contain an Authorization header.
func TestMeEndpoint_MissingToken(t *testing.T) {
	tokenService := &fakeTokenService{}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	meHandler := handlers.NewMeHandler()

	protectedHandler := authMiddleware.RequireAuthentication(
		meHandler,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	response := httptest.NewRecorder()

	protectedHandler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}

	if tokenService.validateCalled {
		t.Fatal(
			"expected token validation not to occur when Authorization header is missing",
		)
	}
}

// TestMeEndpoint_InvalidToken verifies that an invalid access token is
// rejected before the protected handler executes.
func TestMeEndpoint_InvalidToken(t *testing.T) {
	tokenService := &fakeTokenService{
		validateErr: errors.New("invalid token"),
	}

	authMiddleware := middleware.NewAuthenticationMiddleware(
		tokenService,
	)

	meHandler := handlers.NewMeHandler()

	protectedHandler := authMiddleware.RequireAuthentication(
		meHandler,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer invalid-access-token",
	)

	response := httptest.NewRecorder()

	protectedHandler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}
}

// fakeTokenService is a test implementation of the TokenService port.
//
// The test does not need the real JWT implementation because the purpose
// here is to verify the HTTP authentication boundary and protected route
// behavior. JWT cryptographic behavior is already covered by the dedicated
// JWT security tests.
type fakeTokenService struct {
	identity       ports.AuthenticatedIdentity
	validateErr    error
	validateCalled bool
}

func (f *fakeTokenService) GenerateAccessToken(
	ctx context.Context,
	userID user.UserID,
	role user.Role,
) (string, error) {
	return "test-access-token", nil
}

func (f *fakeTokenService) ValidateAccessToken(
	ctx context.Context,
	token string,
) (ports.AuthenticatedIdentity, error) {
	f.validateCalled = true

	if f.validateErr != nil {
		return ports.AuthenticatedIdentity{}, f.validateErr
	}

	return f.identity, nil
}