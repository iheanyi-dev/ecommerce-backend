package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware_GeneratesRequestIDWhenMissing(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
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
		downstreamRequestID,
		recorder.Header().Get(middleware.RequestIDHeader),
	)
}

func TestRequestIDMiddleware_PreservesValidRequestID(t *testing.T) {
	requestID := uuid.NewString()

	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
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
	assert.Equal(
		t,
		requestID,
		recorder.Header().Get(middleware.RequestIDHeader),
	)
}

func TestRequestIDMiddleware_ReplacesInvalidRequestID(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)
	request.Header.Set(
		middleware.RequestIDHeader,
		"invalid-request-id",
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
		"invalid-request-id",
		downstreamRequestID,
	)

	_, err := uuid.Parse(downstreamRequestID)
	require.NoError(t, err)

	assert.Equal(
		t,
		downstreamRequestID,
		recorder.Header().Get(middleware.RequestIDHeader),
	)
}
