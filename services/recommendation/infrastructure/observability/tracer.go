package observability

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/application/ports"
)

var (
	// ErrEmptyTracerName indicates that the tracer has no usable name.
	ErrEmptyTracerName = errors.New("tracer name cannot be empty")
)

// OpenTelemetryTracer implements the application tracing port.
type OpenTelemetryTracer struct {
	tracer trace.Tracer
}

// Compile-time contract verification.
var _ ports.Tracer = (*OpenTelemetryTracer)(nil)

// NewOpenTelemetryTracer creates a tracer and configures W3C trace propagation.
func NewOpenTelemetryTracer(serviceName string) (*OpenTelemetryTracer, error) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil, ErrEmptyTracerName
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &OpenTelemetryTracer{
		tracer: otel.Tracer(serviceName),
	}, nil
}

// Start creates a span while preserving the supplied trace context.
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

	spanContext, span := t.tracer.Start(ctx, name)

	return spanContext, otelSpan{span: span}
}

// otelSpan adapts an OpenTelemetry span to the application Span port.
type otelSpan struct {
	span trace.Span
}

// End ends the underlying OpenTelemetry span.
func (s otelSpan) End() {
	if s.span != nil {
		s.span.End()
	}
}

// noopSpan safely represents an unavailable or invalid tracing span.
type noopSpan struct{}

// End performs no operation.
func (noopSpan) End() {}

// CreateProductionTracer creates the Recommendation service tracer.
func CreateProductionTracer(serviceName string) (*OpenTelemetryTracer, error) {
	return NewOpenTelemetryTracer(serviceName)
}
