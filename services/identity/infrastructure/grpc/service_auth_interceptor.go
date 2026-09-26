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

// NewServiceAuthInterceptor creates a unary gRPC interceptor that authenticates
// trusted internal services using the same credentials as the HTTP service
// authentication middleware.
func NewServiceAuthInterceptor(
	expectedServiceName string,
	expectedServiceSecret string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		serviceName, serviceSecret, ok := serviceCredentials(ctx)

		if !ok ||
			expectedServiceName == "" ||
			expectedServiceSecret == "" ||
			serviceName != expectedServiceName ||
			!secureStringEqual(serviceSecret, expectedServiceSecret) {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid service credentials",
			)
		}

		return handler(ctx, req)
	}
}

func serviceCredentials(ctx context.Context) (string, string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", false
	}

	serviceNames := md.Get(serviceNameMetadata)
	serviceSecrets := md.Get(serviceSecretMetadata)

	if len(serviceNames) == 0 || len(serviceSecrets) == 0 {
		return "", "", false
	}

	return serviceNames[0], serviceSecrets[0], true
}

func secureStringEqual(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}

	leftBytes := []byte(left)
	rightBytes := []byte(right)

	if len(leftBytes) != len(rightBytes) {
		return false
	}

	return subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}
