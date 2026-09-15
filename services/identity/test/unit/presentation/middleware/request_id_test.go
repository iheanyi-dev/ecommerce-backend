package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/presentation/middleware"
)

func TestRequestIDMiddleware_GeneratesRequestIDWhenMissing(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)

	var downstreamRequestID string

	handler := middleware.NewRequestIDMiddleware().Middleware(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			downstreamRequestID = r.Header.Get(
				middleware.RequestIDHeader,
			)
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.NotEmpty(t, downstreamRequestID)

	_, err := uuid.Parse(downstreamRequestID)
	require.NoError(t, err)

	assert.Equal(
		t,
		http.StatusNoContent,
		recorder.Code,
	)

	// Identity establishes the request ID for downstream processing.
	// The client-facing response header is owned by the Gateway.
	assert.Empty(
		t,
		recorder.Header().Get(middleware.RequestIDHeader),
	)
}

func TestRequestIDMiddleware_PreservesValidRequestID(t *testing.T) {
	requestID := uuid.NewString()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)
	request.Header.Set(
		middleware.RequestIDHeader,
		requestID,
	)

	var downstreamRequestID string

	handler := middleware.NewRequestIDMiddleware().Middleware(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			downstreamRequestID = r.Header.Get(
				middleware.RequestIDHeader,
			)
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, requestID, downstreamRequestID)

	// Identity does not duplicate the client-facing request ID.
	assert.Empty(
		t,
		recorder.Header().Get(middleware.RequestIDHeader),
	)
}

func TestRequestIDMiddleware_ReplacesInvalidRequestID(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me",
		nil,
	)
	request.Header.Set(
		middleware.RequestIDHeader,
		"not-a-valid-request-id",
	)

	var downstreamRequestID string

	handler := middleware.NewRequestIDMiddleware().Middleware(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			downstreamRequestID = r.Header.Get(
				middleware.RequestIDHeader,
			)
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.NotEqual(
		t,
		"not-a-valid-request-id",
		downstreamRequestID,
	)

	_, err := uuid.Parse(downstreamRequestID)
	require.NoError(t, err)

	assert.Empty(
		t,
		recorder.Header().Get(middleware.RequestIDHeader),
	)
}
