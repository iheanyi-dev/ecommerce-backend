package grpcserver

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
		if strings.TrimSpace(expectedName) == "" || expectedSecret == "" {
			return nil, status.Error(
				codes.Internal,
				"service authentication is not configured",
			)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(
				codes.Unauthenticated,
				"service credentials are required",
			)
		}

		names := md.Get(serviceNameMetadata)
		secrets := md.Get(serviceSecretMetadata)

		if len(names) != 1 || len(secrets) != 1 {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid service credentials",
			)
		}

		nameOK := subtle.ConstantTimeCompare(
			[]byte(names[0]),
			[]byte(expectedName),
		)
		secretOK := subtle.ConstantTimeCompare(
			[]byte(secrets[0]),
			[]byte(expectedSecret),
		)

		if nameOK != 1 || secretOK != 1 {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid service credentials",
			)
		}

		return handler(ctx, req)
	}
}
