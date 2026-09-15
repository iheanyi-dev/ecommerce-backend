package observability

import (
	"context"
	"errors"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrEmptyTracerName = errors.New("tracer name cannot be empty")
)

// OpenTelemetryTracer is the infrastructure implementation of the
// application-layer Tracer port.
//
// The application layer depends only on ports.Tracer. OpenTelemetry
// remains an infrastructure concern.
type OpenTelemetryTracer struct {
	tracer trace.Tracer
}

// NewOpenTelemetryTracer creates the OpenTelemetry tracer used by the
// Identity service.
//
// The W3C Trace Context propagator is also registered globally so that
// HTTP middleware can extract incoming traceparent/tracestate headers
// and future outbound clients can inject the same context.
func NewOpenTelemetryTracer(
	serviceName string,
) (*OpenTelemetryTracer, error) {
	serviceName = strings.TrimSpace(serviceName)

	if serviceName == "" {
		return nil, ErrEmptyTracerName
	}

	// Register the standard W3C Trace Context propagator.
	//
	// This enables distributed trace context propagation through the
	// standard traceparent and tracestate HTTP headers.
	otel.SetTextMapPropagator(
		propagation.TraceContext{},
	)

	return &OpenTelemetryTracer{
		tracer: otel.Tracer(serviceName),
	}, nil
}

// Start creates a new tracing span.
//
// The supplied context is preserved so parent trace information can
// flow into the newly created span.
func (t *OpenTelemetryTracer) Start(
	ctx context.Context,
	name string,
) (context.Context, ports.Span) {
	if t == nil || t.tracer == nil {
		return ctx, noopSpan{}
	}

	name = strings.TrimSpace(name)

	if name == "" {
		return ctx, noopSpan{}
	}

	spanCtx, span := t.tracer.Start(ctx, name)

	return spanCtx, otelSpan{
		span: span,
	}
}

// otelSpan adapts an OpenTelemetry span to the application-layer
// ports.Span interface.
type otelSpan struct {
	span trace.Span
}

func (s otelSpan) End() {
	if s.span == nil {
		return
	}

	s.span.End()
}

// noopSpan safely represents a span when tracing is unavailable.
type noopSpan struct{}

func (noopSpan) End() {}
