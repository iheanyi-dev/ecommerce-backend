package observability_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/observability"
)

func TestNewOpenTelemetryTracer(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("promotion")
	if err != nil {
		t.Fatalf("NewOpenTelemetryTracer() error = %v", err)
	}
	if tracer == nil {
		t.Fatal("NewOpenTelemetryTracer() returned nil")
	}
}

func TestNewOpenTelemetryTracerRejectsEmptyName(t *testing.T) {
	_, err := observability.NewOpenTelemetryTracer("  ")
	if err != observability.ErrEmptyTracerName {
		t.Fatalf("expected ErrEmptyTracerName, got %v", err)
	}
}

func TestOpenTelemetryTracerStart(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("promotion")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	spanCtx, span := tracer.Start(ctx, "promotion.create")

	if span == nil {
		t.Fatal("Start() returned nil span")
	}
	if spanCtx == nil {
		t.Fatal("Start() returned nil context")
	}

	span.End()
}

func TestOpenTelemetryTracerStartWithEmptySpanName(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("promotion")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	spanCtx, span := tracer.Start(ctx, " ")

	if spanCtx != ctx {
		t.Fatal("expected original context for empty span name")
	}
	if span == nil {
		t.Fatal("expected no-op span")
	}

	span.End()
}
