package grpc_test

import (
	"context"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	identitygrpc "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/grpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockTracer struct {
	startCalled bool
	startCtx    context.Context
	startName   string
	span        ports.Span
}

func (m *mockTracer) Start(
	ctx context.Context,
	name string,
) (context.Context, ports.Span) {
	m.startCalled = true
	m.startCtx = ctx
	m.startName = name

	if m.span == nil {
		m.span = mockSpan{}
	}

	return ctx, m.span
}

var _ ports.Tracer = (*mockTracer)(nil)

type mockSpan struct {
	endCalled bool
}

func (s mockSpan) End() {
	s.endCalled = true
}

func TestTracingInterceptor_ExtractsTraceContext(t *testing.T) {
	tracer := &mockTracer{}

	originalPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTextMapPropagator(originalPropagator)
	})

	traceProvider := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		_ = traceProvider.Shutdown(context.Background())
	})

	originalTracerProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(traceProvider)
	t.Cleanup(func() {
		otel.SetTracerProvider(originalTracerProvider)
	})

	parentCtx, parentSpan := traceProvider.Tracer("test").Start(
		context.Background(),
		"client",
	)
	defer parentSpan.End()

	md := metadata.MD{}
	otel.GetTextMapPropagator().Inject(
		parentCtx,
		metadataCarrier{metadata: md},
	)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		md,
	)

	interceptor := identitygrpc.NewTracingInterceptor(tracer)

	_, err := interceptor(
		ctx,
		nil,
		&grpc.UnaryServerInfo{
			FullMethod: "/identity.v1.IdentityService/GetUser",
		},
		func(handlerCtx context.Context, req interface{}) (interface{}, error) {
			if !oteltrace.SpanContextFromContext(handlerCtx).IsValid() {
				t.Fatal("expected valid trace context in handler")
			}

			return nil, nil
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !tracer.startCalled {
		t.Fatal("expected tracer Start to be called")
	}

	if tracer.startName != "/identity.v1.IdentityService/GetUser" {
		t.Fatalf(
			"expected span name %q, got %q",
			"/identity.v1.IdentityService/GetUser",
			tracer.startName,
		)
	}

	if !oteltrace.SpanContextFromContext(tracer.startCtx).IsValid() {
		t.Fatal("expected extracted trace context to reach tracer")
	}
}

type metadataCarrier struct {
	metadata metadata.MD
}

func (c metadataCarrier) Get(key string) string {
	values := c.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (c metadataCarrier) Set(key string, value string) {
	c.metadata.Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.metadata))
	for key := range c.metadata {
		keys = append(keys, key)
	}

	return keys
}

var _ propagation.TextMapCarrier = metadataCarrier{}
