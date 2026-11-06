package observability

import (
	"context"
	"errors"
	"testing"
)

func TestNewOpenTelemetryTracerRejectsEmptyServiceName(t *testing.T) {
	_, err := NewOpenTelemetryTracer("  ")
	if !errors.Is(err, ErrEmptyTracerName) {
		t.Fatalf("expected ErrEmptyTracerName, got %v", err)
	}
}

func TestOpenTelemetryTracerStartsAndEndsSpan(t *testing.T) {
	tracer, err := NewOpenTelemetryTracer("billing-test")
	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	ctx := context.Background()
	spanContext, span := tracer.Start(ctx, "billing.operation")
	if spanContext == nil {
		t.Fatal("expected a context")
	}
	if span == nil {
		t.Fatal("expected a span")
	}

	span.End()
}

func TestOpenTelemetryTracerUsesNoopSpanForEmptyName(t *testing.T) {
	tracer, err := NewOpenTelemetryTracer("billing-test")
	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	ctx := context.Background()
	resultContext, span := tracer.Start(ctx, " ")

	if resultContext != ctx {
		t.Fatal("expected the original context")
	}
	if span == nil {
		t.Fatal("expected a no-op span")
	}

	span.End()
}

func TestNilOpenTelemetryTracerIsSafe(t *testing.T) {
	var tracer *OpenTelemetryTracer

	ctx := context.Background()
	resultContext, span := tracer.Start(ctx, "billing.operation")

	if resultContext != ctx {
		t.Fatal("expected the original context")
	}
	span.End()
}
