package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/observability"
)

func TestNewOpenTelemetryTracer_RejectsEmptyServiceName(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("   ")

	require.ErrorIs(t, err, observability.ErrEmptyTracerName)
	require.Nil(t, tracer)
}

func TestNewOpenTelemetryTracer_CreatesTracer(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("storage")

	require.NoError(t, err)
	require.NotNil(t, tracer)
}

func TestOpenTelemetryTracer_StartCreatesSpan(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("storage")
	require.NoError(t, err)

	ctx := context.Background()

	spanCtx, span := tracer.Start(ctx, "storage.put_object")

	require.NotNil(t, span)
	require.NotNil(t, spanCtx)

	span.End()
}

func TestOpenTelemetryTracer_EmptySpanNameIsSafe(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("storage")
	require.NoError(t, err)

	ctx := context.Background()

	spanCtx, span := tracer.Start(ctx, "   ")

	require.NotNil(t, span)
	require.Equal(t, ctx, spanCtx)

	span.End()
}

func TestOpenTelemetryTracer_NilReceiverIsSafe(t *testing.T) {
	var tracer *observability.OpenTelemetryTracer

	ctx := context.Background()

	spanCtx, span := tracer.Start(ctx, "storage.operation")

	require.Equal(t, ctx, spanCtx)
	require.NotNil(t, span)

	span.End()
}
