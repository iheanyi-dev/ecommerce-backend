// Package storage implements the Product service's Storage gRPC adapter.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	storagev1 "github.com/iheanyi-dev/ecommerce-backend/proto/storage/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/shared/config"
)

const uploadChunkSize = 32 * 1024

// Client implements ports.StorageClient.
type Client struct {
	conn          *grpc.ClientConn
	client        storagev1.StorageServiceClient
	serviceName   string
	serviceSecret string
}

var _ ports.StorageClient = (*Client)(nil)

// NewClient creates the Storage gRPC client and registers connection cleanup.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("storage client: config is required")
	}
	if cfg.StorageServiceAddr == "" {
		return nil, errors.New("storage client: STORAGE_SERVICE_ADDR is required")
	}
	if cfg.ServiceAuthName == "" || cfg.ServiceAuthSecret == "" {
		return nil, errors.New("storage client: service authentication configuration is required")
	}

	conn, err := grpc.NewClient(
		cfg.StorageServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create Storage gRPC connection: %w", err)
	}

	client := NewClientWithStub(
		storagev1.NewStorageServiceClient(conn),
		cfg.ServiceAuthName,
		cfg.ServiceAuthSecret,
	)
	client.conn = conn

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})
	return client, nil
}

// NewClientWithStub supports deterministic tests using an injected gRPC stub.
func NewClientWithStub(
	client storagev1.StorageServiceClient,
	serviceName string,
	serviceSecret string,
) *Client {
	return &Client{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
	}
}

// PutObject uploads content using the caller-provided reference.
// The metadata message is sent first, followed by bounded-size content chunks.
func (c *Client) PutObject(
	ctx context.Context,
	reference string,
	contentType string,
	content io.Reader,
) error {
	if reference == "" {
		return errors.New("storage: object reference is required")
	}
	if contentType == "" {
		return errors.New("storage: content type is required")
	}
	if content == nil {
		return errors.New("storage: content reader is required")
	}

	ctx = c.authenticatedContext(ctx)
	stream, err := c.client.PutObject(ctx)
	if err != nil {
		return fmt.Errorf("open Storage upload stream: %w", err)
	}

	if err := stream.Send(&storagev1.PutObjectRequest{
		Payload: &storagev1.PutObjectRequest_Metadata{
			Metadata: &storagev1.PutObjectMetadata{
				Reference:   reference,
				ContentType: contentType,
			},
		},
	}); err != nil {
		return fmt.Errorf("send Storage upload metadata: %w", err)
	}

	buffer := make([]byte, uploadChunkSize)
	for {
		n, readErr := content.Read(buffer)
		if n > 0 {
			chunk := append([]byte(nil), buffer[:n]...)
			if err := stream.Send(&storagev1.PutObjectRequest{
				Payload: &storagev1.PutObjectRequest_Data{Data: chunk},
			}); err != nil {
				return fmt.Errorf("send Storage upload content: %w", err)
			}
		}

		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read Storage upload content: %w", readErr)
		}
	}

	if _, err := stream.CloseAndRecv(); err != nil {
		return fmt.Errorf("finish Storage upload: %w", err)
	}
	return nil
}

// DeleteObject requests deletion of the exact supplied storage reference.
func (c *Client) DeleteObject(ctx context.Context, reference string) error {
	if reference == "" {
		return errors.New("storage: object reference is required")
	}

	ctx = c.authenticatedContext(ctx)
	_, err := c.client.DeleteObject(ctx, &storagev1.DeleteObjectRequest{
		Reference: reference,
	})
	if err != nil {
		return fmt.Errorf("delete Storage object %q: %w", reference, err)
	}
	return nil
}

// authenticatedContext adds service authentication and propagates trace context.
func (c *Client) authenticatedContext(ctx context.Context) context.Context {
	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	md, _ := metadata.FromOutgoingContext(ctx)
	if md == nil {
		md = metadata.MD{}
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(md))
	return metadata.NewOutgoingContext(ctx, md)
}
