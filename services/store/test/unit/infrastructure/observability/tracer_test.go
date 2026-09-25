package observability_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/observability"
)

func TestOpenTelemetryTracer_StartAndEnd(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("store")
	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	ctx := context.Background()

	spanCtx, span := tracer.Start(ctx, "store.create")

	if span == nil {
		t.Fatal("expected span, got nil")
	}

	if spanCtx == nil {
		t.Fatal("expected context, got nil")
	}

	span.End()
}

func TestOpenTelemetryTracer_RejectsEmptyName(t *testing.T) {
	_, err := observability.NewOpenTelemetryTracer("")

	if err != observability.ErrEmptyTracerName {
		t.Fatalf(
			"expected ErrEmptyTracerName, got %v",
			err,
		)
	}
}

func TestOpenTelemetryTracer_EmptySpanNameIsSafe(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("store")
	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	ctx := context.Background()

	spanCtx, span := tracer.Start(ctx, "")

	if span == nil {
		t.Fatal("expected noop span, got nil")
	}

	if spanCtx != ctx {
		t.Fatal("expected original context for empty span name")
	}

	span.End()
}
