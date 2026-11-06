package grpc

import (
	"context"
	"crypto/subtle"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	serviceNameMetadata   = "x-service-name"
	serviceSecretMetadata = "x-service-secret"
)

func NewServiceAuthInterceptor(expectedName, expectedSecret string) grpc.UnaryServerInterceptor {
	expectedName = strings.TrimSpace(expectedName)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if expectedName == "" || expectedSecret == "" {
			return nil, status.Error(codes.Unauthenticated, "service authentication is not configured")
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "service credentials are required")
		}

		names := md.Get(serviceNameMetadata)
		secrets := md.Get(serviceSecretMetadata)
		if len(names) != 1 || len(secrets) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid service credentials")
		}

		name := strings.TrimSpace(names[0])
		secret := secrets[0]

		if subtle.ConstantTimeCompare([]byte(name), []byte(expectedName)) != 1 ||
			subtle.ConstantTimeCompare([]byte(secret), []byte(expectedSecret)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid service credentials")
		}

		return handler(ctx, req)
	}
}
