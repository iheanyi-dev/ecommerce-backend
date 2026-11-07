package middleware

import (
	"net/http"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

// Observability records HTTP request metrics and logs.
type Observability struct {
	logger  ports.Logger
	metrics ports.Metrics
}

// NewObservability creates HTTP observability middleware.
func NewObservability(
	logger ports.Logger,
	metrics ports.Metrics,
) *Observability {
	return &Observability{
		logger:  logger,
		metrics: metrics,
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}

func (m *Observability) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		ctx := r.Context()
		duration := time.Since(start).Seconds()

		if m.metrics != nil {
			_ = m.metrics.Increment(ctx, ports.Metric{
				Name:  "recommendation.http.requests",
				Value: 1,
			})
			_ = m.metrics.Observe(ctx, ports.Metric{
				Name:  "recommendation.http.request.duration",
				Value: duration,
			})
		}

		if m.logger != nil {
			_ = m.logger.Log(ctx, ports.LogEvent{
				Event:     "recommendation.http.request_completed",
				Operation: "http_request",
			})
		}
	})
}
