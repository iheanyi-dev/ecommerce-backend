package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/handlers"
)

type mockUpdateUserStatusService struct {
	called bool

	userID  string
	command dto.UpdateUserStatusCommand

	result dto.UpdateUserStatusResult
	err    error
}

func (m *mockUpdateUserStatusService) Execute(
	ctx context.Context,
	userID string,
	command dto.UpdateUserStatusCommand,
) (dto.UpdateUserStatusResult, error) {
	m.called = true
	m.userID = userID
	m.command = command

	if m.err != nil {
		return dto.UpdateUserStatusResult{}, m.err
	}

	return m.result, nil
}

var _ ports.UpdateUserStatusService = (*mockUpdateUserStatusService)(nil)

func TestUpdateUserStatusHandler_Success(t *testing.T) {
	service := &mockUpdateUserStatusService{
		result: dto.UpdateUserStatusResult{
			UserID: "user-123",
			Status: "suspended",
		},
	}

	handler := handlers.NewUpdateUserStatusHandler(service)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/admin/users/user-123/status",
		strings.NewReader(`{"status":"suspended"}`),
	)
	request.SetPathValue("id", "user-123")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d",
			http.StatusOK,
			response.Code,
		)
	}

	if !service.called {
		t.Fatal("expected service to be called")
	}

	if service.userID != "user-123" {
		t.Fatalf("expected user ID %q, got %q",
			"user-123",
			service.userID,
		)
	}

	if service.command.Status != "suspended" {
		t.Fatalf("expected status %q, got %q",
			"suspended",
			service.command.Status,
		)
	}

	var body struct {
		UserID string `json:"user_id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.UserID != "user-123" {
		t.Fatalf("expected response user ID %q, got %q",
			"user-123",
			body.UserID,
		)
	}

	if body.Status != "suspended" {
		t.Fatalf("expected response status %q, got %q",
			"suspended",
			body.Status,
		)
	}
}

func TestUpdateUserStatusHandler_RejectsInvalidJSON(t *testing.T) {
	service := &mockUpdateUserStatusService{}
	handler := handlers.NewUpdateUserStatusHandler(service)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/admin/users/user-123/status",
		strings.NewReader(`{"status":`),
	)
	request.SetPathValue("id", "user-123")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf(
			"expected content type %q, got %q",
			"application/json",
			response.Header().Get("Content-Type"),
		)
	}

	expectedBody := `{"error":"invalid request body"}` + "\n"

	if response.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %q, got %q",
			expectedBody,
			response.Body.String(),
		)
	}

	if service.called {
		t.Fatal("expected service not to be called")
	}
}

func TestUpdateUserStatusHandler_RejectsUnsupportedMethod(t *testing.T) {
	service := &mockUpdateUserStatusService{}
	handler := handlers.NewUpdateUserStatusHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/users/user-123/status",
		nil,
	)
	request.SetPathValue("id", "user-123")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d",
			http.StatusMethodNotAllowed,
			response.Code,
		)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf(
			"expected content type %q, got %q",
			"application/json",
			response.Header().Get("Content-Type"),
		)
	}

	expectedBody := `{"error":"method not allowed"}` + "\n"

	if response.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %q, got %q",
			expectedBody,
			response.Body.String(),
		)
	}

	if service.called {
		t.Fatal("expected service not to be called")
	}
}

func TestUpdateUserStatusHandler_ServiceError(t *testing.T) {
	service := &mockUpdateUserStatusService{
		err: errors.New("status update failed"),
	}

	handler := handlers.NewUpdateUserStatusHandler(service)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/admin/users/user-123/status",
		strings.NewReader(`{"status":"suspended"}`),
	)
	request.SetPathValue("id", "user-123")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d",
			http.StatusInternalServerError,
			response.Code,
		)
	}
}

func TestUpdateUserStatusHandler_RejectsMissingUserID(t *testing.T) {
	service := &mockUpdateUserStatusService{}
	handler := handlers.NewUpdateUserStatusHandler(service)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/admin/users//status",
		strings.NewReader(`{"status":"suspended"}`),
	)

	// Deliberately do not set the "id" path value.
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}

	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf(
			"expected content type %q, got %q",
			"application/json",
			response.Header().Get("Content-Type"),
		)
	}

	expectedBody := `{"error":"user id is required"}` + "\n"

	if response.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %q, got %q",
			expectedBody,
			response.Body.String(),
		)
	}

	if service.called {
		t.Fatal("expected service not to be called")
	}
}
