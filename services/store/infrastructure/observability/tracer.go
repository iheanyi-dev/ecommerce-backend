package observability

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
)

var (
	// ErrEmptyTracerName indicates that the tracer was created without a
	// usable service/instrumentation name.
	ErrEmptyTracerName = errors.New("tracer name cannot be empty")
)

// OpenTelemetryTracer is the Infrastructure implementation of the Store
// application-level ports.Tracer contract.
//
// The application layer knows only about ports.Tracer. OpenTelemetry remains
// entirely inside Infrastructure.
type OpenTelemetryTracer struct {
	tracer trace.Tracer
}

// Compile-time contract verification.
var _ ports.Tracer = (*OpenTelemetryTracer)(nil)

// NewOpenTelemetryTracer creates the Store OpenTelemetry tracer.
//
// The standard W3C Trace Context propagator is registered so incoming
// traceparent/tracestate context can participate in distributed tracing.
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
//
// The incoming context is preserved so an existing parent span can flow
// into the new Store span.
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

// otelSpan adapts an OpenTelemetry span to the application-level Span port.
type otelSpan struct {
	span trace.Span
}

func (s otelSpan) End() {
	if s.span == nil {
		return
	}

	s.span.End()
}

// noopSpan safely represents an unavailable or invalid tracing span.
type noopSpan struct{}

func (noopSpan) End() {}
