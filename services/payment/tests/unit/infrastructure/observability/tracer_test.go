package observability_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/observability"
)

func TestNewOpenTelemetryTracer(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("payment")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tracer == nil {
		t.Fatal("expected tracer")
	}
}

func TestNewOpenTelemetryTracerRejectsEmptyName(t *testing.T) {
	_, err := observability.NewOpenTelemetryTracer("   ")

	if !errors.Is(err, observability.ErrEmptyTracerName) {
		t.Fatalf("expected ErrEmptyTracerName, got %v", err)
	}
}

func TestOpenTelemetryTracerStart(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("payment")

	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	type contextKey string

	ctx := context.WithValue(
		context.Background(),
		contextKey("test"),
		"value",
	)

	spanCtx, span := tracer.Start(ctx, "payment.test")

	if spanCtx == nil {
		t.Fatal("expected span context")
	}

	if span == nil {
		t.Fatal("expected span")
	}

	if spanCtx.Value(contextKey("test")) != "value" {
		t.Fatal("expected incoming context values to be preserved")
	}

	span.End()
}

func TestOpenTelemetryTracerStartWithEmptySpanName(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("payment")

	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	ctx := context.Background()

	spanCtx, span := tracer.Start(ctx, "   ")

	if spanCtx != ctx {
		t.Fatal("expected original context for empty span name")
	}

	if span == nil {
		t.Fatal("expected no-op span")
	}

	span.End()
}

func TestNilOpenTelemetryTracerStart(t *testing.T) {
	var tracer *observability.OpenTelemetryTracer

	ctx := context.Background()

	spanCtx, span := tracer.Start(ctx, "payment.test")

	if spanCtx != ctx {
		t.Fatal("expected original context")
	}

	if span == nil {
		t.Fatal("expected no-op span")
	}

	span.End()
}
