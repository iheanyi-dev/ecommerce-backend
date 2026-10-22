package middleware

import (
	"net/http"
	"strconv"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
)

type TracingMiddleware struct {
	tracer ports.Tracer
}

func NewTracingMiddleware(tracer ports.Tracer) *TracingMiddleware {
	return &TracingMiddleware{tracer: tracer}
}

func (m *TracingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := m.tracer.Start(r.Context(), "cart.http")
		defer span.End()

		span.SetAttribute("http.method", r.Method)
		span.SetAttribute("http.target", r.URL.Path)

		recorder := &statusResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r.WithContext(ctx))

		span.SetIntAttribute("http.status_code", recorder.status)
		if recorder.status >= http.StatusInternalServerError {
			span.SetErrorStatus("HTTP " + strconv.Itoa(recorder.status))
		}
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// Flush preserves streaming support when the underlying writer supports it.
func (w *statusResponseWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap allows http.ResponseController to reach the underlying writer.
func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
