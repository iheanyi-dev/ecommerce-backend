package grpc

import (
	"context"
	"errors"
	"io"

	"github.com/iheanyi-dev/ecommerce-backend/proto/storage/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/dto"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/storage/infrastructure/persistence/filesystem"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the internal StorageService gRPC contract.
//
// The server is deliberately an adapter around application service ports.
// It does not contain storage business logic or filesystem logic.
type Server struct {
	storagev1.UnimplementedStorageServiceServer

	putObject        ports.PutObjectService
	getPrivateObject ports.GetPrivateObjectService
	deleteObject     ports.DeleteObjectService
}

// NewServer constructs the Storage gRPC server.
func NewServer(
	putObject ports.PutObjectService,
	getPrivateObject ports.GetPrivateObjectService,
	deleteObject ports.DeleteObjectService,
) *Server {
	return &Server{
		putObject:        putObject,
		getPrivateObject: getPrivateObject,
		deleteObject:     deleteObject,
	}
}

// Compile-time assertion that Server implements the generated contract.
var _ storagev1.StorageServiceServer = (*Server)(nil)

// PutObject receives object metadata followed by streamed content chunks.
//
// The first message must contain metadata. Content is passed through an
// io.Pipe so the application layer can consume it incrementally without
// buffering the complete object in memory.
func (s *Server) PutObject(
	stream storagev1.StorageService_PutObjectServer,
) error {
	ctx := stream.Context()

	var metadata *storagev1.PutObjectMetadata

	for {
		request, err := stream.Recv()
		if err == io.EOF {
			if metadata == nil {
				return status.Error(
					codes.InvalidArgument,
					"object metadata is required",
				)
			}

			// Metadata without content is a valid empty object.
			return s.putEmptyObject(ctx, metadata)
		}
		if err != nil {
			return mapGRPCError(err)
		}

		if request == nil {
			return status.Error(
				codes.InvalidArgument,
				"put object request is required",
			)
		}

		switch payload := request.Payload.(type) {
		case *storagev1.PutObjectRequest_Metadata:
			if metadata != nil {
				return status.Error(
					codes.InvalidArgument,
					"object metadata may only be sent once",
				)
			}

			if payload.Metadata == nil {
				return status.Error(
					codes.InvalidArgument,
					"object metadata is required",
				)
			}

			metadata = payload.Metadata

			// Once metadata has been received, hand the remainder of the
			// gRPC stream to the application through an io.Pipe.
			return s.putStreamedObject(stream, metadata)

		case *storagev1.PutObjectRequest_Data:
			// Content before metadata is intentionally rejected. This makes
			// the stream protocol deterministic and prevents ambiguous
			// object creation.
			return status.Error(
				codes.InvalidArgument,
				"object metadata must be sent before content",
			)

		default:
			return status.Error(
				codes.InvalidArgument,
				"put object payload is required",
			)
		}
	}
}

// putStreamedObject bridges the gRPC receive stream to the application's
// io.Reader contract.
func (s *Server) putStreamedObject(
	stream storagev1.StorageService_PutObjectServer,
	metadata *storagev1.PutObjectMetadata,
) error {
	ctx := stream.Context()

	reader, writer := io.Pipe()

	result := make(chan error, 1)

	go func() {
		result <- s.putObject.Execute(
			ctx,
			dto.PutObjectInput{
				Reference:   metadata.GetReference(),
				Content:     reader,
				ContentType: metadata.GetContentType(),
			},
		)
	}()

	for {
		request, err := stream.Recv()

		if err == io.EOF {
			if closeErr := writer.Close(); closeErr != nil {
				return mapGRPCError(closeErr)
			}

			if executeErr := <-result; executeErr != nil {
				return mapGRPCError(executeErr)
			}

			return stream.SendAndClose(&storagev1.PutObjectResponse{})
		}

		if err != nil {
			_ = writer.CloseWithError(err)
			<-result

			return mapGRPCError(err)
		}

		if request == nil {
			streamErr := status.Error(
				codes.InvalidArgument,
				"put object request is required",
			)

			_ = writer.CloseWithError(streamErr)
			<-result

			return streamErr
		}

		switch payload := request.Payload.(type) {
		case *storagev1.PutObjectRequest_Data:
			if _, err := writer.Write(payload.Data); err != nil {
				<-result
				return mapGRPCError(err)
			}

		case *storagev1.PutObjectRequest_Metadata:
			streamErr := status.Error(
				codes.InvalidArgument,
				"object metadata may only be sent once",
			)

			_ = writer.CloseWithError(streamErr)
			<-result

			return streamErr

		default:
			streamErr := status.Error(
				codes.InvalidArgument,
				"put object payload is required",
			)

			_ = writer.CloseWithError(streamErr)
			<-result

			return streamErr
		}
	}
}

// putEmptyObject handles a metadata-only stream.
func (s *Server) putEmptyObject(
	ctx context.Context,
	metadata *storagev1.PutObjectMetadata,
) error {
	err := s.putObject.Execute(
		ctx,
		dto.PutObjectInput{
			Reference:   metadata.GetReference(),
			Content:     &emptyReader{},
			ContentType: metadata.GetContentType(),
		},
	)
	if err != nil {
		return mapGRPCError(err)
	}

	return status.Error(codes.OK, "")
}

// GetPrivateObject retrieves an object and streams it to the caller.
func (s *Server) GetPrivateObject(
	request *storagev1.GetPrivateObjectRequest,
	stream storagev1.StorageService_GetPrivateObjectServer,
) error {
	if request == nil {
		return status.Error(
			codes.InvalidArgument,
			"get private object request is required",
		)
	}

	reference := request.GetReference()
	if reference == "" {
		return status.Error(
			codes.InvalidArgument,
			"object reference is required",
		)
	}

	output, err := s.getPrivateObject.Execute(
		stream.Context(),
		dto.GetPrivateObjectInput{
			Reference: reference,
		},
	)
	if err != nil {
		return mapGRPCError(err)
	}

	defer output.Content.Close()

	buffer := make([]byte, 32*1024)
	first := true

	for {
		n, readErr := output.Content.Read(buffer)

		if n > 0 {
			response := &storagev1.GetPrivateObjectResponse{
				Data: buffer[:n],
			}

			if first {
				response.ContentType = output.ContentType
				first = false
			}

			if err := stream.Send(response); err != nil {
				return mapGRPCError(err)
			}
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			return mapGRPCError(readErr)
		}
	}

	// Ensure the caller receives metadata even for an empty object.
	if first {
		if err := stream.Send(&storagev1.GetPrivateObjectResponse{
			ContentType: output.ContentType,
		}); err != nil {
			return mapGRPCError(err)
		}
	}

	return nil
}

// DeleteObject deletes an object through the application boundary.
func (s *Server) DeleteObject(
	ctx context.Context,
	request *storagev1.DeleteObjectRequest,
) (*storagev1.DeleteObjectResponse, error) {
	if request == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"delete object request is required",
		)
	}

	reference := request.GetReference()
	if reference == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"object reference is required",
		)
	}

	if err := s.deleteObject.Execute(
		ctx,
		dto.DeleteObjectInput{
			Reference: reference,
		},
	); err != nil {
		return nil, mapGRPCError(err)
	}

	return &storagev1.DeleteObjectResponse{}, nil
}

// emptyReader represents a valid empty object body.
type emptyReader struct{}

func (*emptyReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

// mapGRPCError translates infrastructure/application errors into stable
// transport-level gRPC status codes.
func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		if status.Code(err) != codes.Unknown {
			return err
		}
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")

	case errors.Is(err, filesystem.ErrInvalidReference):
		return status.Error(codes.InvalidArgument, "invalid object reference")

	case errors.Is(err, filesystem.ErrObjectNotFound):
		return status.Error(codes.NotFound, "object not found")

	default:
		return status.Error(codes.Internal, "internal storage error")
	}
}

// Keep the imported gRPC package explicitly tied to the generated server
// contract and avoid accidentally depending on a different transport.
var _ gogrpc.ServiceRegistrar = (*gogrpc.Server)(nil)
