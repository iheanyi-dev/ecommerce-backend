package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/presentation/middleware"
)

func TestRequestIDMiddlewarePreservesValidRequestID(t *testing.T) {
	requestID := uuid.NewString()
	called := false

	handler := middleware.NewRequestIDMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			if got := r.Header.Get(middleware.RequestIDHeader); got != requestID {
				t.Fatalf("expected request ID %q in request, got %q", requestID, got)
			}
		}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(middleware.RequestIDHeader, requestID)

	handler.ServeHTTP(recorder, request)

	if !called {
		t.Fatal("expected next handler to be called")
	}
	if got := recorder.Header().Get(middleware.RequestIDHeader); got != requestID {
		t.Fatalf("expected response request ID %q, got %q", requestID, got)
	}
}

func TestRequestIDMiddlewareGeneratesIDWhenMissing(t *testing.T) {
	var requestID string

	handler := middleware.NewRequestIDMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID = r.Header.Get(middleware.RequestIDHeader)
		}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(recorder, request)

	if _, err := uuid.Parse(requestID); err != nil {
		t.Fatalf("expected generated UUID, got %q: %v", requestID, err)
	}
	if got := recorder.Header().Get(middleware.RequestIDHeader); got != requestID {
		t.Fatalf("expected response request ID %q, got %q", requestID, got)
	}
}

func TestRequestIDMiddlewareReplacesInvalidRequestID(t *testing.T) {
	var requestID string

	handler := middleware.NewRequestIDMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID = r.Header.Get(middleware.RequestIDHeader)
		}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(middleware.RequestIDHeader, "invalid-request-id")

	handler.ServeHTTP(recorder, request)

	if _, err := uuid.Parse(requestID); err != nil {
		t.Fatalf("expected replacement UUID, got %q: %v", requestID, err)
	}
	if requestID == "invalid-request-id" {
		t.Fatal("expected invalid request ID to be replaced")
	}
	if got := recorder.Header().Get(middleware.RequestIDHeader); got != requestID {
		t.Fatalf("expected response request ID %q, got %q", requestID, got)
	}
}

func TestRequestIDMiddlewareTrimsRequestID(t *testing.T) {
	requestID := uuid.NewString()
	var gotRequestID string

	handler := middleware.NewRequestIDMiddleware().Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotRequestID = r.Header.Get(middleware.RequestIDHeader)
		}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(middleware.RequestIDHeader, "  "+requestID+"  ")

	handler.ServeHTTP(recorder, request)

	if gotRequestID != requestID {
		t.Fatalf("expected trimmed request ID %q, got %q", requestID, gotRequestID)
	}
}
