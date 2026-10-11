package storage

import (
	"bytes"
	"context"
	"errors"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	storagev1 "github.com/iheanyi-dev/ecommerce-backend/proto/storage/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/shared/config"
	"go.uber.org/fx"
)

const uploadChunkSize = 32 * 1024

type Client struct {
	conn          *grpc.ClientConn
	client        storagev1.StorageServiceClient
	serviceName   string
	serviceSecret string
	tracer        ports.Tracer
}

func NewClient(
	lc fx.Lifecycle,
	cfg *config.Config,
	tracer ports.Tracer,
) (*Client, error) {
	conn, err := grpc.NewClient(
		cfg.StorageServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := NewClientWithStub(
		storagev1.NewStorageServiceClient(conn),
		cfg.ServiceAuthName,
		cfg.ServiceAuthSecret,
		tracer,
	)

	client.conn = conn

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return conn.Close()
		},
	})

	return client, nil
}

// NewClientWithStub creates a Storage client around an injected gRPC
// client stub. It is primarily useful for deterministic infrastructure tests.
func NewClientWithStub(
	client storagev1.StorageServiceClient,
	serviceName string,
	serviceSecret string,
	tracer ports.Tracer,
) *Client {
	return &Client{
		client:        client,
		serviceName:   serviceName,
		serviceSecret: serviceSecret,
		tracer:        tracer,
	}
}

func (c *Client) PutStoreImage(
	ctx context.Context,
	image ports.StoreImage,
) error {
	if image.Reference == "" {
		return errors.New("storage: image reference is required")
	}
	return c.putObject(ctx, image.Reference, image.ContentType, image.Content)
}

// DeleteKnowledgeFile requests deletion of a Knowledge file from Storage.
func (c *Client) DeleteKnowledgeFile(ctx context.Context, reference string) error {
	if reference == "" {
		return errors.New("storage: knowledge reference is required")
	}

	ctx, span := c.tracer.Start(ctx, "storage.DeleteObject")
	defer span.End()

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(md))
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.client.DeleteObject(ctx, &storagev1.DeleteObjectRequest{
		Reference: reference,
	})
	return err
}

// PutKnowledgeFile synchronously uploads a Knowledge file to Storage.
func (c *Client) PutKnowledgeFile(
	ctx context.Context,
	file ports.KnowledgeFile,
) error {
	if file.Reference == "" {
		return errors.New("storage: knowledge reference is required")
	}
	return c.putObject(ctx, file.Reference, file.ContentType, file.Content)
}

func (c *Client) putObject(
	ctx context.Context,
	reference string,
	contentType string,
	content []byte,
) error {
	ctx, span := c.tracer.Start(ctx, "storage.PutObject")
	defer span.End()

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-service-name", c.serviceName,
		"x-service-secret", c.serviceSecret,
	)

	// Inject the current W3C trace context into gRPC metadata.
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	otel.GetTextMapPropagator().Inject(
		ctx,
		propagation.HeaderCarrier(md),
	)

	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := c.client.PutObject(ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(&storagev1.PutObjectRequest{
		Payload: &storagev1.PutObjectRequest_Metadata{
			Metadata: &storagev1.PutObjectMetadata{
				Reference:   reference,
				ContentType: contentType,
			},
		},
	}); err != nil {
		return err
	}

	reader := bytes.NewReader(content)
	buffer := make([]byte, uploadChunkSize)

	for {
		n, readErr := reader.Read(buffer)

		if n > 0 {
			chunk := append([]byte(nil), buffer[:n]...)

			if err := stream.Send(&storagev1.PutObjectRequest{
				Payload: &storagev1.PutObjectRequest_Data{
					Data: chunk,
				},
			}); err != nil {
				return err
			}
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			return readErr
		}
	}

	_, err = stream.CloseAndRecv()
	return err
}
