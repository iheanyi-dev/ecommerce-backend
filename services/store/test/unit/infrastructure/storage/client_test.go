package storage_test

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	storagev1 "github.com/iheanyi-dev/ecommerce-backend/proto/storage/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	storageinfra "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/storage"
)

const (
	testServiceName   = "store"
	testServiceSecret = "store-storage-test-secret"
)

type storageTestServer struct {
	storagev1.UnimplementedStorageServiceServer

	serviceName   string
	serviceSecret string

	requests []*storagev1.PutObjectRequest
	putError error
}

func (s *storageTestServer) PutObject(
	stream grpc.ClientStreamingServer[storagev1.PutObjectRequest, storagev1.PutObjectResponse],
) error {
	ctx := stream.Context()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}

	names := md.Get("x-service-name")
	secrets := md.Get("x-service-secret")

	if len(names) != 1 || len(secrets) != 1 ||
		names[0] != s.serviceName ||
		secrets[0] != s.serviceSecret {
		return status.Error(codes.Unauthenticated, "invalid service credentials")
	}

	for {
		request, err := stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		s.requests = append(s.requests, request)
	}

	if s.putError != nil {
		return s.putError
	}

	return stream.SendAndClose(&storagev1.PutObjectResponse{})
}

func newStorageTestClient(
	t *testing.T,
	server storagev1.StorageServiceServer,
) storagev1.StorageServiceClient {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)

	grpcServer := grpc.NewServer()
	storagev1.RegisterStorageServiceServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	return storagev1.NewStorageServiceClient(conn)
}

func TestImageStorage_PutStoreImage_SendsAuthenticatedStream(t *testing.T) {
	server := &storageTestServer{
		serviceName:   testServiceName,
		serviceSecret: testServiceSecret,
	}

	client := newStorageTestClient(t, server)

	adapter := newProductionImageStorage(t, client)

	storeID := uuid.New()

	err := adapter.PutStoreImage(
		context.Background(),
		ports.StoreImage{
			StoreID:     storeID,
			Reference:   "stores/" + storeID.String() + ".png",
			Content:     []byte("image-content"),
			ContentType: "image/png",
		},
	)

	require.NoError(t, err)
	require.Len(t, server.requests, 2)

	metadataRequest := server.requests[0].GetMetadata()
	require.NotNil(t, metadataRequest)
	require.Equal(
		t,
		"stores/"+storeID.String()+".png",
		metadataRequest.Reference,
	)
	require.Equal(t, "image/png", metadataRequest.ContentType)

	require.Equal(
		t,
		[]byte("image-content"),
		server.requests[1].GetData(),
	)
}

func TestImageStorage_PutStoreImage_SendsEmptyContentAfterMetadata(t *testing.T) {
	server := &storageTestServer{
		serviceName:   testServiceName,
		serviceSecret: testServiceSecret,
	}

	client := newStorageTestClient(t, server)

	adapter := newProductionImageStorage(t, client)

	storeID := uuid.New()

	err := adapter.PutStoreImage(
		context.Background(),
		ports.StoreImage{
			StoreID:     storeID,
			Reference:   "stores/" + storeID.String() + ".png",
			Content:     nil,
			ContentType: "image/png",
		},
	)

	require.NoError(t, err)
	require.Len(t, server.requests, 1)
	require.NotNil(t, server.requests[0].GetMetadata())
}

func TestImageStorage_PutStoreImage_PropagatesStorageError(t *testing.T) {
	expectedErr := status.Error(
		codes.Internal,
		"storage unavailable",
	)

	server := &storageTestServer{
		serviceName:   testServiceName,
		serviceSecret: testServiceSecret,
		putError:      expectedErr,
	}

	client := newStorageTestClient(t, server)

	adapter := newProductionImageStorage(t, client)

	storeID := uuid.New()

	err := adapter.PutStoreImage(
		context.Background(),
		ports.StoreImage{
			StoreID:     storeID,
			Reference:   "stores/" + storeID.String() + ".png",
			Content:     []byte("image-content"),
			ContentType: "image/png",
		},
	)

	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
	require.Equal(t, "storage unavailable", status.Convert(err).Message())
}

func TestImageStorage_PutStoreImage_RejectsEmptyReference(t *testing.T) {
	server := &storageTestServer{
		serviceName:   testServiceName,
		serviceSecret: testServiceSecret,
	}

	client := newStorageTestClient(t, server)

	adapter := newProductionImageStorage(t, client)

	err := adapter.PutStoreImage(
		context.Background(),
		ports.StoreImage{
			StoreID:     uuid.New(),
			Reference:   "",
			Content:     []byte("image-content"),
			ContentType: "image/png",
		},
	)

	require.Error(t, err)
	require.Empty(t, server.requests)
}

func TestImageStorage_PutStoreImage_PropagatesContextCancellation(t *testing.T) {
	server := &storageTestServer{
		serviceName:   testServiceName,
		serviceSecret: testServiceSecret,
		putError:      status.Error(codes.Canceled, "request canceled"),
	}

	client := newStorageTestClient(t, server)

	adapter := newProductionImageStorage(t, client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := adapter.PutStoreImage(
		ctx,
		ports.StoreImage{
			StoreID:     uuid.New(),
			Reference:   "stores/test.png",
			Content:     []byte("image-content"),
			ContentType: "image/png",
		},
	)

	require.Error(t, err)
}

type testTracer struct{}

type testSpan struct{}

func (testTracer) Start(ctx context.Context, _ string) (context.Context, ports.Span) {
	return ctx, testSpan{}
}

func (testSpan) End() {}

func newProductionImageStorage(
	t *testing.T,
	client storagev1.StorageServiceClient,
) *storageinfra.Client {
	t.Helper()

	return storageinfra.NewClientWithStub(
		client,
		testServiceName,
		testServiceSecret,
		testTracer{},
	)
}
