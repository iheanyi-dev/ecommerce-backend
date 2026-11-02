package presentation_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/presentation/middleware"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		incomingRequestID string
	}{
		{
			name:              "preserves valid request id",
			incomingRequestID: uuid.NewString(),
		},
		{
			name:              "replaces missing request id",
			incomingRequestID: "",
		},
		{
			name:              "replaces invalid request id",
			incomingRequestID: "not-a-uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := middleware.NewRequestIDMiddleware()

			var received string

			handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received = r.Header.Get(middleware.RequestIDHeader)
				w.WriteHeader(http.StatusNoContent)
			}))

			request := httptest.NewRequest(http.MethodGet, "/api/v1/stores", nil)

			if tt.incomingRequestID != "" {
				request.Header.Set(
					middleware.RequestIDHeader,
					tt.incomingRequestID,
				)
			}

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNoContent, recorder.Code)
			require.NotEmpty(t, received)
			require.Equal(
				t,
				received,
				recorder.Header().Get(middleware.RequestIDHeader),
			)

			_, err := uuid.Parse(received)
			require.NoError(t, err)

			if tt.incomingRequestID != "" {
				parsedIncoming, err := uuid.Parse(tt.incomingRequestID)

				if err == nil {
					require.Equal(t, parsedIncoming.String(), received)
				}
			}
		})
	}
}
