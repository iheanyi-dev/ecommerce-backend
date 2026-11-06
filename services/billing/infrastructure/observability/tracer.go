package observability

import (
	"errors"
	"strings"

	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var ErrEmptyTracerName = errors.New("tracer service name cannot be empty")

// OpenTelemetryTracer adapts OpenTelemetry to the application tracing port.
type OpenTelemetryTracer struct {
	tracer trace.Tracer
}

// NewOpenTelemetryTracer creates a service-scoped OpenTelemetry tracer.
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

// Start begins a span while preserving the supplied parent context.
func (tracer *OpenTelemetryTracer) Start(
	ctx context.Context,
	name string,
) (context.Context, ports.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	name = strings.TrimSpace(name)
	if tracer == nil || tracer.tracer == nil || name == "" {
		return ctx, noopSpan{}
	}

	ctx, span := tracer.tracer.Start(ctx, name)
	return ctx, otelSpan{span: span}
}

type otelSpan struct {
	span trace.Span
}

func (span otelSpan) End() {
	if span.span != nil {
		span.span.End()
	}
}

type noopSpan struct{}

func (noopSpan) End() {}

var _ ports.Tracer = (*OpenTelemetryTracer)(nil)
