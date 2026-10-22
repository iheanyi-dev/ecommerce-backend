package store_test

import (
	"context"
	"net"
	"testing"

	storev1 "github.com/iheanyi-dev/ecommerce-backend/proto/store/v1"
	productstore "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/store"

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

type testStoreServer struct {
	storev1.UnimplementedStoreServiceServer

	t                     *testing.T
	ownerID               string
	storeID               string
	active                bool
	activeStoreIDs        []string
	incrementErr          error
	decrementErr          error
	getOwnerErr           error
	isActiveErr           error
	filterActiveStoresErr error
}

func (s *testStoreServer) checkAuth(ctx context.Context) error {
	s.t.Helper()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}
	if got := md.Get("x-service-name"); len(got) != 1 || got[0] != testServiceName {
		return status.Error(codes.Unauthenticated, "invalid service name")
	}
	if got := md.Get("x-service-secret"); len(got) != 1 || got[0] != testServiceSecret {
		return status.Error(codes.Unauthenticated, "invalid service secret")
	}
	return nil
}

func (s *testStoreServer) GetStoreIDByOwnerID(
	ctx context.Context,
	req *storev1.GetStoreIDByOwnerIDRequest,
) (*storev1.GetStoreIDByOwnerIDResponse, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if s.getOwnerErr != nil {
		return nil, s.getOwnerErr
	}
	if req.GetOwnerId() != s.ownerID {
		return nil, status.Error(codes.NotFound, "owner has no store")
	}
	return &storev1.GetStoreIDByOwnerIDResponse{StoreId: s.storeID}, nil
}

func (s *testStoreServer) IncrementProductCount(
	ctx context.Context,
	req *storev1.IncrementProductCountRequest,
) (*storev1.IncrementProductCountResponse, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if s.incrementErr != nil {
		return nil, s.incrementErr
	}
	if req.GetStoreId() != s.storeID {
		return nil, status.Error(codes.NotFound, "store not found")
	}
	return &storev1.IncrementProductCountResponse{}, nil
}

func (s *testStoreServer) DecrementProductCount(
	ctx context.Context,
	req *storev1.DecrementProductCountRequest,
) (*storev1.DecrementProductCountResponse, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if s.decrementErr != nil {
		return nil, s.decrementErr
	}
	if req.GetStoreId() != s.storeID {
		return nil, status.Error(codes.NotFound, "store not found")
	}
	return &storev1.DecrementProductCountResponse{}, nil
}

func (s *testStoreServer) IsStoreActive(
	ctx context.Context,
	req *storev1.IsStoreActiveRequest,
) (*storev1.IsStoreActiveResponse, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if s.isActiveErr != nil {
		return nil, s.isActiveErr
	}
	if req.GetStoreId() != s.storeID {
		return nil, status.Error(codes.NotFound, "store not found")
	}
	return &storev1.IsStoreActiveResponse{Active: s.active}, nil
}

func (s *testStoreServer) FilterActiveStoreIDs(
	ctx context.Context,
	req *storev1.FilterActiveStoreIDsRequest,
) (*storev1.FilterActiveStoreIDsResponse, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if s.filterActiveStoresErr != nil {
		return nil, s.filterActiveStoresErr
	}
	return &storev1.FilterActiveStoreIDsResponse{
		ActiveStoreIds: s.activeStoreIDs,
	}, nil
}

func newTestClient(t *testing.T, server *testStoreServer) *productstore.Client {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	storev1.RegisterStoreServiceServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		grpcServer.Stop()
		_ = listener.Close()
		t.Fatalf("create test gRPC connection: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	})

	return productstore.NewClientWithStub(
		storev1.NewStoreServiceClient(conn),
		testServiceName,
		testServiceSecret,
	)
}

func TestClientStoreOperations(t *testing.T) {
	server := &testStoreServer{
		t:              t,
		ownerID:        "owner-1",
		storeID:        "store-1",
		active:         true,
		activeStoreIDs: []string{"store-1", "store-3"},
	}
	client := newTestClient(t, server)
	ctx := context.Background()

	t.Run("gets store ID by owner", func(t *testing.T) {
		got, err := client.GetStoreIDByOwnerID(ctx, "owner-1")
		if err != nil {
			t.Fatalf("GetStoreIDByOwnerID() error = %v", err)
		}
		if got != "store-1" {
			t.Fatalf("GetStoreIDByOwnerID() = %q, want %q", got, "store-1")
		}
	})

	t.Run("owner without a store returns empty ID", func(t *testing.T) {
		got, err := client.GetStoreIDByOwnerID(ctx, "owner-without-store")
		if err != nil {
			t.Fatalf("GetStoreIDByOwnerID() error = %v", err)
		}
		if got != "" {
			t.Fatalf("GetStoreIDByOwnerID() = %q, want empty ID", got)
		}
	})

	t.Run("reserves product capacity", func(t *testing.T) {
		if err := client.IncrementProductCount(ctx, "store-1"); err != nil {
			t.Fatalf("IncrementProductCount() error = %v", err)
		}
	})

	t.Run("releases product capacity", func(t *testing.T) {
		if err := client.DecrementProductCount(ctx, "store-1"); err != nil {
			t.Fatalf("DecrementProductCount() error = %v", err)
		}
	})

	t.Run("checks store active status", func(t *testing.T) {
		active, err := client.IsStoreActive(ctx, "store-1")
		if err != nil {
			t.Fatalf("IsStoreActive() error = %v", err)
		}
		if !active {
			t.Fatal("IsStoreActive() = false, want true")
		}
	})

	t.Run("filters active store IDs", func(t *testing.T) {
		got, err := client.FilterActiveStoreIDs(ctx, []string{"store-1", "store-2", "store-3"})
		if err != nil {
			t.Fatalf("FilterActiveStoreIDs() error = %v", err)
		}
		if len(got) != 2 || got[0] != "store-1" || got[1] != "store-3" {
			t.Fatalf("FilterActiveStoreIDs() = %v, want [store-1 store-3]", got)
		}
	})

	t.Run("empty filter avoids an RPC", func(t *testing.T) {
		got, err := client.FilterActiveStoreIDs(ctx, nil)
		if err != nil {
			t.Fatalf("FilterActiveStoreIDs() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("FilterActiveStoreIDs() = %v, want empty result", got)
		}
	})
}

func TestClientPropagatesStoreErrors(t *testing.T) {
	t.Run("capacity exhaustion is preserved", func(t *testing.T) {
		server := &testStoreServer{
			t:            t,
			storeID:      "store-1",
			incrementErr: status.Error(codes.ResourceExhausted, "product limit reached"),
		}
		err := newTestClient(t, server).IncrementProductCount(context.Background(), "store-1")
		if status.Code(err) != codes.ResourceExhausted {
			t.Fatalf("status.Code(err) = %v, want ResourceExhausted; error: %v", status.Code(err), err)
		}
	})

	t.Run("owner lookup dependency failure is preserved", func(t *testing.T) {
		server := &testStoreServer{
			t:           t,
			getOwnerErr: status.Error(codes.Unavailable, "Store unavailable"),
		}
		_, err := newTestClient(t, server).GetStoreIDByOwnerID(context.Background(), "owner-1")
		if status.Code(err) != codes.Unavailable {
			t.Fatalf("status.Code(err) = %v, want Unavailable; error: %v", status.Code(err), err)
		}
	})

	t.Run("capacity release failure is returned", func(t *testing.T) {
		server := &testStoreServer{
			t:            t,
			storeID:      "store-1",
			decrementErr: status.Error(codes.Unavailable, "Store unavailable"),
		}
		err := newTestClient(t, server).DecrementProductCount(context.Background(), "store-1")
		if status.Code(err) != codes.Unavailable {
			t.Fatalf("status.Code(err) = %v, want Unavailable; error: %v", status.Code(err), err)
		}
	})
}
