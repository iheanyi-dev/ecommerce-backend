package observability

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/application/ports"
)

var (
	ErrEmptyTracerName = errors.New("tracer name cannot be empty")
)

// OpenTelemetryTracer implements the Payment application tracer port.
type OpenTelemetryTracer struct {
	tracer trace.Tracer
}

var _ ports.Tracer = (*OpenTelemetryTracer)(nil)

// NewOpenTelemetryTracer creates the Payment OpenTelemetry tracer.
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

// Start creates an application tracing span.
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

type otelSpan struct {
	span trace.Span
}

func (s otelSpan) End() {
	if s.span == nil {
		return
	}

	s.span.End()
}

type noopSpan struct{}

func (noopSpan) End() {}
