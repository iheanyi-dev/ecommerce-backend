package middleware

import (
	"net/http"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/gateway/application/ports"
)

// RequestObservabilityMiddleware records the final outcome of every Gateway
// HTTP request.
//
// Observability is intentionally best-effort: logging failures must never
// change the HTTP response or interrupt request processing.
type RequestObservabilityMiddleware struct {
	logger ports.Logger
}

// NewRequestObservabilityMiddleware creates Gateway request observability
// middleware.
func NewRequestObservabilityMiddleware(
	logger ports.Logger,
) *RequestObservabilityMiddleware {
	return &RequestObservabilityMiddleware{
		logger: logger,
	}
}

// Middleware records the HTTP method, path, status, duration, and request ID
// after the downstream handler has completed.
//
// The middleware does not log request bodies, authorization credentials,
// service secrets, or other sensitive request data.
func (m *RequestObservabilityMiddleware) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		start := time.Now()

		recorder := newResponseRecorder(w)

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)

		m.logger.Info(
			r.Context(),
			"http.request.completed",
			ports.Field{
				Key:   "http.method",
				Value: r.Method,
			},
			ports.Field{
				Key:   "http.path",
				Value: r.URL.Path,
			},
			ports.Field{
				Key:   "http.status_code",
				Value: recorder.statusCode,
			},
			ports.Field{
				Key:   "http.duration_ms",
				Value: duration.Milliseconds(),
			},
			ports.Field{
				Key:   "request_id",
				Value: r.Header.Get(RequestIDHeader),
			},
		)
	})
}

// responseRecorder captures the final HTTP status while preserving the
// underlying ResponseWriter behavior needed by Gateway handlers and proxies.
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.statusCode = statusCode
	r.wroteHeader = true

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.ResponseWriter.Write(body)
}
