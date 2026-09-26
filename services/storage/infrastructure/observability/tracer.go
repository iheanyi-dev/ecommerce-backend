package observability

import (
	"context"
	"errors"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrEmptyTracerName = errors.New("tracer name cannot be empty")
)

// OpenTelemetryTracer is the Infrastructure implementation of the
// application-level ports.Tracer contract.
//
// The application layer depends only on ports.Tracer. OpenTelemetry remains
// isolated inside Infrastructure.
type OpenTelemetryTracer struct {
	tracer trace.Tracer
}

// NewOpenTelemetryTracer creates the OpenTelemetry tracer used by the
// Storage service.
//
// The standard W3C Trace Context propagator is registered so incoming and
// outgoing distributed trace context can use traceparent and tracestate.
func NewOpenTelemetryTracer(
	serviceName string,
) (*OpenTelemetryTracer, error) {
	serviceName = strings.TrimSpace(serviceName)

	if serviceName == "" {
		return nil, ErrEmptyTracerName
	}

	otel.SetTextMapPropagator(
		propagation.TraceContext{},
	)

	return &OpenTelemetryTracer{
		tracer: otel.Tracer(serviceName),
	}, nil
}

// Start creates a new tracing span while preserving the supplied parent
// context.
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

// otelSpan adapts an OpenTelemetry span to the application-level ports.Span.
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
