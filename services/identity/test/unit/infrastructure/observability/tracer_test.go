package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/observability"
)

func TestNewOpenTelemetryTracer_RejectsEmptyServiceName(t *testing.T) {
	_, err := observability.NewOpenTelemetryTracer("   ")

	require.ErrorIs(t, err, observability.ErrEmptyTracerName)
}

func TestNewOpenTelemetryTracer_CreatesTracer(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("identity")

	require.NoError(t, err)
	require.NotNil(t, tracer)
}

func TestOpenTelemetryTracer_StartCreatesSpan(t *testing.T) {
	provider := trace.NewTracerProvider()
	previousProvider := otel.GetTracerProvider()

	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		_ = provider.Shutdown(context.Background())
	})

	tracer, err := observability.NewOpenTelemetryTracer("identity")
	require.NoError(t, err)

	ctx, span := tracer.Start(
		context.Background(),
		"identity.test",
	)

	require.NotNil(t, ctx)
	require.NotNil(t, span)

	spanFromContext := oteltrace.SpanFromContext(ctx)

	require.True(t, spanFromContext.SpanContext().IsValid())

	span.End()
}

func TestOpenTelemetryTracer_StartWithEmptySpanNameDoesNotCreateSpan(t *testing.T) {
	provider := trace.NewTracerProvider()
	previousProvider := otel.GetTracerProvider()

	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		_ = provider.Shutdown(context.Background())
	})

	tracer, err := observability.NewOpenTelemetryTracer("identity")
	require.NoError(t, err)

	ctx, span := tracer.Start(
		context.Background(),
		"   ",
	)

	require.Equal(t, context.Background(), ctx)
	require.NotNil(t, span)

	span.End()

	require.False(
		t,
		oteltrace.SpanFromContext(ctx).SpanContext().IsValid(),
	)
}

func TestOpenTelemetryTracer_NilTracerIsSafe(t *testing.T) {
	var tracer *observability.OpenTelemetryTracer

	ctx, span := tracer.Start(
		context.Background(),
		"identity.test",
	)

	require.Equal(t, context.Background(), ctx)
	require.NotNil(t, span)

	require.NotPanics(t, func() {
		span.End()
	})
}

func TestOpenTelemetryTracer_StartPreservesParentTrace(t *testing.T) {
	provider := trace.NewTracerProvider()
	previousProvider := otel.GetTracerProvider()

	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		_ = provider.Shutdown(context.Background())
	})

	tracer, err := observability.NewOpenTelemetryTracer("identity")
	require.NoError(t, err)

	parentCtx, parentSpan := tracer.Start(
		context.Background(),
		"identity.parent",
	)
	defer parentSpan.End()

	childCtx, childSpan := tracer.Start(
		parentCtx,
		"identity.child",
	)
	defer childSpan.End()

	parentSpanContext := oteltrace.SpanFromContext(parentCtx).SpanContext()
	childSpanContext := oteltrace.SpanFromContext(childCtx).SpanContext()

	require.True(t, parentSpanContext.IsValid())
	require.True(t, childSpanContext.IsValid())

	require.Equal(
		t,
		parentSpanContext.TraceID(),
		childSpanContext.TraceID(),
	)
}
func TestNewOpenTelemetryTracer_ConfiguresW3CTraceContextPropagator(
	t *testing.T,
) {
	previousPropagator := otel.GetTextMapPropagator()

	t.Cleanup(func() {
		otel.SetTextMapPropagator(previousPropagator)
	})

	_, err := observability.NewOpenTelemetryTracer("identity")

	require.NoError(t, err)

	propagator := otel.GetTextMapPropagator()

	require.IsType(
		t,
		propagation.TraceContext{},
		propagator,
	)
}
