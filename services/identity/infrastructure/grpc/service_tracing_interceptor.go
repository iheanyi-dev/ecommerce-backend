package grpc

import (
	"context"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type tracingMetadataCarrier struct {
	metadata metadata.MD
}

func (c tracingMetadataCarrier) Get(key string) string {
	values := c.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (c tracingMetadataCarrier) Set(key string, value string) {
	c.metadata.Set(key, value)
}

func (c tracingMetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.metadata))

	for key := range c.metadata {
		keys = append(keys, key)
	}

	return keys
}

var _ propagation.TextMapCarrier = tracingMetadataCarrier{}

// NewTracingInterceptor creates a unary gRPC interceptor that extracts
// W3C Trace Context from incoming gRPC metadata and starts a server span.
func NewTracingInterceptor(
	tracer ports.Tracer,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if incomingMetadata, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = otel.GetTextMapPropagator().Extract(
				ctx,
				tracingMetadataCarrier{
					metadata: incomingMetadata,
				},
			)
		}

		if tracer == nil {
			return handler(ctx, req)
		}

		spanName := ""
		if info != nil {
			spanName = info.FullMethod
		}

		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		return handler(ctx, req)
	}
}
