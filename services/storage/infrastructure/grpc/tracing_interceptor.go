package grpc

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"

	"go.opentelemetry.io/otel"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// NewTracingInterceptor creates the unary tracing interceptor.
//
// Incoming W3C trace context is extracted from gRPC metadata before the
// Storage span is started. The application receives the resulting context.
func NewTracingInterceptor(tracer ports.Tracer) gogrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *gogrpc.UnaryServerInfo,
		handler gogrpc.UnaryHandler,
	) (interface{}, error) {
		if tracer == nil {
			return handler(ctx, req)
		}

		ctx = extractTraceContext(ctx)

		ctx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

		return handler(ctx, req)
	}
}

// NewStreamTracingInterceptor creates the streaming tracing interceptor.
//
// gRPC streaming handlers obtain their context through ServerStream.Context(),
// so the stream is wrapped with the traced context before invoking the
// generated service implementation.
func NewStreamTracingInterceptor(
	tracer ports.Tracer,
) gogrpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream gogrpc.ServerStream,
		info *gogrpc.StreamServerInfo,
		handler gogrpc.StreamHandler,
	) error {
		if tracer == nil {
			return handler(srv, stream)
		}

		ctx := extractTraceContext(stream.Context())

		ctx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

		return handler(
			srv,
			&tracedServerStream{
				ServerStream: stream,
				ctx:          ctx,
			},
		)
	}
}

// tracedServerStream overrides Context so downstream streaming handlers use
// the traced context instead of the original transport context.
type tracedServerStream struct {
	gogrpc.ServerStream
	ctx context.Context
}

func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}

// extractTraceContext extracts W3C Trace Context from incoming gRPC metadata.
//
// The propagator is kept at the transport boundary so the application layer
// remains independent of OpenTelemetry.
func extractTraceContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	carrier := metadataCarrier(md)

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// metadataCarrier adapts gRPC metadata to OpenTelemetry's TextMapCarrier.
type metadataCarrier metadata.MD

func (c metadataCarrier) Get(key string) string {
	values := metadata.MD(c).Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (c metadataCarrier) Set(key string, value string) {
	metadata.MD(c).Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c))

	for key := range c {
		keys = append(keys, key)
	}

	return keys
}

// Keep the OpenTelemetry propagation imports explicitly referenced by this
// adapter so changes to the project's tracing implementation remain obvious.
// var (
// 	_ propagation.TextMapCarrier = metadataCarrier{}
// 	_ propagation.TextMapPropagator
// 	_ propagator.TextMapPropagator
// 	_ stats.Handler
// )
