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

// NewServiceAuthInterceptor creates the unary service-authentication
// interceptor.
//
// Storage is internal-only, so every unary RPC must carry trusted service
// credentials.
func NewServiceAuthInterceptor(
	expectedName string,
	expectedSecret string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !authenticated(ctx, expectedName, expectedSecret) {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid service credentials",
			)
		}

		return handler(ctx, req)
	}
}

// NewServiceAuthStreamInterceptor protects client- and server-streaming RPCs.
func NewServiceAuthStreamInterceptor(
	expectedName string,
	expectedSecret string,
) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !authenticated(
			stream.Context(),
			expectedName,
			expectedSecret,
		) {
			return status.Error(
				codes.Unauthenticated,
				"invalid service credentials",
			)
		}

		return handler(srv, stream)
	}
}

func authenticated(
	ctx context.Context,
	expectedName string,
	expectedSecret string,
) bool {
	if expectedName == "" || expectedSecret == "" {
		return false
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}

	names := md.Get(serviceNameMetadata)
	secrets := md.Get(serviceSecretMetadata)

	if len(names) == 0 || len(secrets) == 0 {
		return false
	}

	if names[0] != expectedName {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(secrets[0]),
		[]byte(expectedSecret),
	) == 1
}
