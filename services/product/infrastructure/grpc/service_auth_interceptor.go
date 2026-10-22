package grpc

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NewServiceAuthInterceptor authenticates internal service-to-service calls.
func NewServiceAuthInterceptor(expectedName, expectedSecret string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if expectedName == "" || expectedSecret == "" {
			return nil, status.Error(codes.Internal, "gRPC service authentication is not configured")
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "service credentials are required")
		}

		names := md.Get("x-service-name")
		secrets := md.Get("x-service-secret")
		if len(names) != 1 || len(secrets) != 1 {
			return nil, status.Error(codes.Unauthenticated, "service credentials are required")
		}

		if names[0] != expectedName ||
			subtle.ConstantTimeCompare([]byte(secrets[0]), []byte(expectedSecret)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid service credentials")
		}

		return handler(ctx, req)
	}
}
