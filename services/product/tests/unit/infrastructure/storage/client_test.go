package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	storagev1 "github.com/iheanyi-dev/ecommerce-backend/proto/storage/v1"
	productstorage "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/storage"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const (
	testServiceName   = "product"
	testServiceSecret = "test-service-secret"
)

type storageServer struct {
	storagev1.UnimplementedStorageServiceServer

	t *testing.T

	uploadPayloads  []storagev1.PutObjectRequest
	uploadMetadata  metadata.MD
	deleteMetadata  metadata.MD
	deleteReference string

	uploadError error
	deleteError error
}

func (s *storageServer) PutObject(
	stream grpc.ClientStreamingServer[storagev1.PutObjectRequest, storagev1.PutObjectResponse],
) error {
	s.uploadMetadata, _ = metadata.FromIncomingContext(stream.Context())

	for {
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		s.uploadPayloads = append(s.uploadPayloads, *request)
	}

	if s.uploadError != nil {
		return s.uploadError
	}

	return stream.SendAndClose(&storagev1.PutObjectResponse{})
}

func (s *storageServer) DeleteObject(
	ctx context.Context,
	request *storagev1.DeleteObjectRequest,
) (*storagev1.DeleteObjectResponse, error) {
	s.deleteMetadata, _ = metadata.FromIncomingContext(ctx)
	s.deleteReference = request.GetReference()

	if s.deleteError != nil {
		return nil, s.deleteError
	}

	return &storagev1.DeleteObjectResponse{}, nil
}

func newTestClient(t *testing.T, serverImpl *storageServer) *productstorage.Client {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	storagev1.RegisterStorageServiceServer(server, serverImpl)

	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		server.Stop()
		_ = listener.Close()
		t.Fatalf("create gRPC client connection: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
	})

	return productstorage.NewClientWithStub(
		storagev1.NewStorageServiceClient(conn),
		testServiceName,
		testServiceSecret,
	)
}

func assertServiceAuthentication(t *testing.T, md metadata.MD) {
	t.Helper()

	if got := md.Get("x-service-name"); len(got) != 1 || got[0] != testServiceName {
		t.Fatalf("x-service-name = %v, want [%q]", got, testServiceName)
	}
	if got := md.Get("x-service-secret"); len(got) != 1 || got[0] != testServiceSecret {
		t.Fatalf("x-service-secret = %v, want configured secret", got)
	}
}

func TestPutObjectSendsMetadataFirstAndPreservesReferenceAndContent(t *testing.T) {
	server := &storageServer{t: t}
	client := newTestClient(t, server)

	reference := "products/product-123.webp"
	contentType := "image/webp"
	content := bytes.Repeat([]byte("product-image-data-"), 5000)

	err := client.PutObject(
		context.Background(),
		reference,
		contentType,
		bytes.NewReader(content),
	)
	if err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}

	if len(server.uploadPayloads) < 2 {
		t.Fatalf("received %d upload payloads, want metadata and data", len(server.uploadPayloads))
	}

	first := server.uploadPayloads[0].GetMetadata()
	if first == nil {
		t.Fatal("first upload payload is not metadata")
	}
	if first.GetReference() != reference {
		t.Errorf("reference = %q, want %q", first.GetReference(), reference)
	}
	if first.GetContentType() != contentType {
		t.Errorf("content type = %q, want %q", first.GetContentType(), contentType)
	}

	var received bytes.Buffer
	for i, request := range server.uploadPayloads[1:] {
		chunk := request.GetData()
		if chunk == nil {
			t.Fatalf("payload %d after metadata is not data", i+1)
		}
		if len(chunk) > 32*1024 {
			t.Errorf("chunk %d has %d bytes; maximum is %d", i, len(chunk), 32*1024)
		}
		_, _ = received.Write(chunk)
	}

	if !bytes.Equal(received.Bytes(), content) {
		t.Errorf("uploaded content does not match input: got %d bytes, want %d", received.Len(), len(content))
	}

	assertServiceAuthentication(t, server.uploadMetadata)
}

func TestDeleteObjectUsesReferenceAndServiceAuthentication(t *testing.T) {
	server := &storageServer{t: t}
	client := newTestClient(t, server)

	reference := "products/product-123.webp"
	if err := client.DeleteObject(context.Background(), reference); err != nil {
		t.Fatalf("DeleteObject() error = %v", err)
	}

	if server.deleteReference != reference {
		t.Errorf("reference = %q, want %q", server.deleteReference, reference)
	}
	assertServiceAuthentication(t, server.deleteMetadata)
}

func TestPutObjectPropagatesStorageServiceError(t *testing.T) {
	serverErr := status.Error(codes.Internal, "storage write failed")
	server := &storageServer{t: t, uploadError: serverErr}
	client := newTestClient(t, server)

	err := client.PutObject(
		context.Background(),
		"products/product-123.webp",
		"image/webp",
		strings.NewReader("image"),
	)
	if err == nil {
		t.Fatal("PutObject() error = nil, want storage service error")
	}
	if status.Code(err) != codes.Internal {
		t.Errorf("error code = %s, want %s (error: %v)", status.Code(err), codes.Internal, err)
	}
}

func TestDeleteObjectPropagatesStorageServiceError(t *testing.T) {
	serverErr := status.Error(codes.Unavailable, "storage unavailable")
	server := &storageServer{t: t, deleteError: serverErr}
	client := newTestClient(t, server)

	err := client.DeleteObject(context.Background(), "products/product-123.webp")
	if err == nil {
		t.Fatal("DeleteObject() error = nil, want storage service error")
	}
	if status.Code(err) != codes.Unavailable {
		t.Errorf("error code = %s, want %s (error: %v)", status.Code(err), codes.Unavailable, err)
	}
}

func TestPutObjectRejectsInvalidArguments(t *testing.T) {
	client := productstorage.NewClientWithStub(nil, testServiceName, testServiceSecret)

	tests := []struct {
		name        string
		reference   string
		contentType string
		content     io.Reader
	}{
		{name: "empty reference", contentType: "image/webp", content: strings.NewReader("image")},
		{name: "empty content type", reference: "products/a.webp", content: strings.NewReader("image")},
		{name: "nil content", reference: "products/a.webp", contentType: "image/webp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.PutObject(context.Background(), tt.reference, tt.contentType, tt.content)
			if err == nil {
				t.Fatal("PutObject() error = nil, want validation error")
			}
		})
	}
}

func TestDeleteObjectRejectsEmptyReference(t *testing.T) {
	client := productstorage.NewClientWithStub(nil, testServiceName, testServiceSecret)

	err := client.DeleteObject(context.Background(), "")
	if err == nil {
		t.Fatal("DeleteObject() error = nil, want validation error")
	}
}
