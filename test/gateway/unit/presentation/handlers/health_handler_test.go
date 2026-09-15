package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	handler := handlers.NewHealthHandler()

	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var response map[string]string

	err := json.NewDecoder(recorder.Body).Decode(&response)

	require.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}
