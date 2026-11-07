package observability_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/iheanyi-dev/ecommerce-backend/services/recommendation/infrastructure/observability"
)

func TestNewOpenTelemetryTracerRejectsEmptyName(t *testing.T) {
	_, err := observability.NewOpenTelemetryTracer(" ")
	if err != observability.ErrEmptyTracerName {
		t.Fatalf("expected ErrEmptyTracerName, got %v", err)
	}
}

func TestNewOpenTelemetryTracerConfiguresW3CPropagation(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("recommendation-test")
	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}
	if tracer == nil {
		t.Fatal("expected tracer instance")
	}

	if _, ok := otel.GetTextMapPropagator().(propagation.TraceContext); !ok {
		t.Fatal("expected W3C TraceContext propagator")
	}
}

func TestOpenTelemetryTracerStartsAndEndsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)

	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(previousProvider)
	})

	tracer, err := observability.NewOpenTelemetryTracer("recommendation-test")
	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	spanContext, span := tracer.Start(context.Background(), "recommendation.operation")

	if span == nil {
		t.Fatal("expected span")
	}
	if !oteltrace.SpanFromContext(spanContext).IsRecording() {
		t.Fatal("expected returned context to contain an active recording span")
	}

	span.End()

	if got := len(exporter.GetSpans()); got != 1 {
		t.Fatalf("expected 1 exported span, got %d", got)
	}
}

func TestOpenTelemetryTracerReturnsNoopSpanForEmptyName(t *testing.T) {
	tracer, err := observability.NewOpenTelemetryTracer("recommendation-test")
	if err != nil {
		t.Fatalf("create tracer: %v", err)
	}

	ctx := context.Background()
	returnedContext, span := tracer.Start(ctx, " ")

	if returnedContext != ctx {
		t.Fatal("expected original context to be preserved")
	}
	if span == nil {
		t.Fatal("expected no-op span")
	}

	span.End()
}

func TestNilOpenTelemetryTracerReturnsNoopSpan(t *testing.T) {
	var tracer *observability.OpenTelemetryTracer

	ctx := context.Background()
	returnedContext, span := tracer.Start(ctx, "recommendation.operation")

	if returnedContext != ctx {
		t.Fatal("expected original context to be preserved")
	}
	if span == nil {
		t.Fatal("expected no-op span")
	}

	span.End()
}
