package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
)

// mockLogoutUserService is a test double for the application-layer
// LogoutUserService port.
//
// The handler must depend only on the application port and must not know
// anything about the concrete logout use case or infrastructure.
type mockLogoutUserService struct {
	err          error
	logoutToken  string
	logoutCalled bool
}

// Logout implements ports.LogoutUserService.
//
// The mock records the supplied refresh token so the test can verify that
// the handler passes the request value unchanged to the application layer.
func (m *mockLogoutUserService) Logout(
	ctx context.Context,
	refreshToken string,
) error {
	m.logoutCalled = true
	m.logoutToken = refreshToken

	return m.err
}

// Compile-time assertion that the mock satisfies the application contract.
var _ ports.LogoutUserService = (*mockLogoutUserService)(nil)

func TestLogoutUserHandler_ServeHTTP_Success(t *testing.T) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Arrange
	// -------------------------------------------------------------------------

	refreshToken := "valid-refresh-token"

	service := &mockLogoutUserService{}

	handler := handlers.NewLogoutUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		strings.NewReader(`{"refresh_token":"valid-refresh-token"}`),
	)

	recorder := httptest.NewRecorder()

	// -------------------------------------------------------------------------
	// Act
	// -------------------------------------------------------------------------

	handler.ServeHTTP(recorder, request)

	// -------------------------------------------------------------------------
	// Assert
	// -------------------------------------------------------------------------

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}

	if !service.logoutCalled {
		t.Fatal("expected Logout to be called")
	}

	if service.logoutToken != refreshToken {
		t.Fatalf(
			"expected refresh token %q, got %q",
			refreshToken,
			service.logoutToken,
		)
	}

	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", recorder.Body.String())
	}
}

func TestLogoutUserHandler_ServeHTTP_RejectsNonPostMethod(t *testing.T) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Arrange
	// -------------------------------------------------------------------------

	service := &mockLogoutUserService{}
	handler := handlers.NewLogoutUserHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/logout",
		nil,
	)

	recorder := httptest.NewRecorder()

	// -------------------------------------------------------------------------
	// Act
	// -------------------------------------------------------------------------

	handler.ServeHTTP(recorder, request)

	// -------------------------------------------------------------------------
	// Assert
	// -------------------------------------------------------------------------

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}

	if service.logoutCalled {
		t.Fatal("expected Logout not to be called")
	}
}

func TestLogoutUserHandler_ServeHTTP_RejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Arrange
	// -------------------------------------------------------------------------

	service := &mockLogoutUserService{}
	handler := handlers.NewLogoutUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		strings.NewReader(`{"refresh_token":`),
	)

	recorder := httptest.NewRecorder()

	// -------------------------------------------------------------------------
	// Act
	// -------------------------------------------------------------------------

	handler.ServeHTTP(recorder, request)

	// -------------------------------------------------------------------------
	// Assert
	// -------------------------------------------------------------------------

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	if service.logoutCalled {
		t.Fatal("expected Logout not to be called")
	}
}

func TestLogoutUserHandler_ServeHTTP_ReturnsUnauthorizedForInvalidToken(
	t *testing.T,
) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Arrange
	// -------------------------------------------------------------------------

	service := &mockLogoutUserService{
		err: use_cases.ErrInvalidRefreshToken,
	}

	handler := handlers.NewLogoutUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		strings.NewReader(`{"refresh_token":"invalid-refresh-token"}`),
	)

	recorder := httptest.NewRecorder()

	// -------------------------------------------------------------------------
	// Act
	// -------------------------------------------------------------------------

	handler.ServeHTTP(recorder, request)

	// -------------------------------------------------------------------------
	// Assert
	// -------------------------------------------------------------------------

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}

	if !service.logoutCalled {
		t.Fatal("expected Logout to be called")
	}
}

func TestLogoutUserHandler_ServeHTTP_ReturnsInternalServerErrorForHashingFailure(
	t *testing.T,
) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Arrange
	// -------------------------------------------------------------------------

	service := &mockLogoutUserService{
		err: use_cases.ErrRefreshTokenHashing,
	}

	handler := handlers.NewLogoutUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		strings.NewReader(`{"refresh_token":"valid-refresh-token"}`),
	)

	recorder := httptest.NewRecorder()

	// -------------------------------------------------------------------------
	// Act
	// -------------------------------------------------------------------------

	handler.ServeHTTP(recorder, request)

	// -------------------------------------------------------------------------
	// Assert
	// -------------------------------------------------------------------------

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}

func TestLogoutUserHandler_ServeHTTP_ReturnsInternalServerErrorForRevocationFailure(
	t *testing.T,
) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Arrange
	// -------------------------------------------------------------------------

	service := &mockLogoutUserService{
		err: use_cases.ErrRefreshTokenRevocation,
	}

	handler := handlers.NewLogoutUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		strings.NewReader(`{"refresh_token":"valid-refresh-token"}`),
	)

	recorder := httptest.NewRecorder()

	// -------------------------------------------------------------------------
	// Act
	// -------------------------------------------------------------------------

	handler.ServeHTTP(recorder, request)

	// -------------------------------------------------------------------------
	// Assert
	// -------------------------------------------------------------------------

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}

func TestLogoutUserHandler_ServeHTTP_ReturnsInternalServerErrorForUnexpectedError(
	t *testing.T,
) {
	t.Parallel()

	// -------------------------------------------------------------------------
	// Arrange
	// -------------------------------------------------------------------------

	service := &mockLogoutUserService{
		err: errors.New("unexpected application failure"),
	}

	handler := handlers.NewLogoutUserHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/users/logout",
		strings.NewReader(`{"refresh_token":"valid-refresh-token"}`),
	)

	recorder := httptest.NewRecorder()

	// -------------------------------------------------------------------------
	// Act
	// -------------------------------------------------------------------------

	handler.ServeHTTP(recorder, request)

	// -------------------------------------------------------------------------
	// Assert
	// -------------------------------------------------------------------------

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}
