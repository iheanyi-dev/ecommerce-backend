package grpc

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	serviceNameMetadata   = "x-service-name"
	serviceSecretMetadata = "x-service-secret"
)

// NewServiceAuthInterceptor authenticates every internal Store RPC.
func NewServiceAuthInterceptor(
	expectedName string,
	expectedSecret string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok || expectedName == "" || expectedSecret == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid service credentials")
		}

		names := md.Get(serviceNameMetadata)
		secrets := md.Get(serviceSecretMetadata)
		if len(names) == 0 || len(secrets) == 0 ||
			names[0] != expectedName ||
			!secureStringEqual(secrets[0], expectedSecret) {
			return nil, status.Error(codes.Unauthenticated, "invalid service credentials")
		}

		return handler(ctx, req)
	}
}

func secureStringEqual(left, right string) bool {
	if left == "" || right == "" || len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
