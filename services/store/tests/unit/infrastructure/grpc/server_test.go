package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	storev1 "github.com/iheanyi-dev/ecommerce-backend/proto/store/v1"
	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/valueobjects"
	storegrpc "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serverStoreRepository struct {
	ports.StoreRepository
	store          *entities.Store
	ownerStore     *entities.Store
	findByIDErr    error
	findByOwnerErr error
}

func (f *serverStoreRepository) FindByID(context.Context, uuid.UUID) (*entities.Store, error) {
	return f.store, f.findByIDErr
}

func (f *serverStoreRepository) FindByOwnerID(context.Context, uuid.UUID) (*entities.Store, error) {
	return f.ownerStore, f.findByOwnerErr
}

type serverProductCapacity struct {
	ports.ProductCapacityRepository
	incrementErr error
	decrementErr error
	incremented  uuid.UUID
	decremented  uuid.UUID
}

func (f *serverProductCapacity) IncrementProductCount(_ context.Context, id uuid.UUID) (int, error) {
	f.incremented = id
	return 1, f.incrementErr
}

func (f *serverProductCapacity) DecrementProductCount(_ context.Context, id uuid.UUID) (int, error) {
	f.decremented = id
	return 0, f.decrementErr
}

type serverActiveStoreLookup struct {
	ports.ActiveStoreLookupRepository
	ids []uuid.UUID
	err error
}

func (f *serverActiveStoreLookup) FilterActiveStoreIDs(_ context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	f.ids = ids
	return f.idsResult(), f.err
}

func (f *serverActiveStoreLookup) idsResult() []uuid.UUID {
	return append([]uuid.UUID(nil), f.ids...)
}

func newServerStore(t *testing.T, ownerID uuid.UUID) *entities.Store {
	t.Helper()
	store, err := entities.NewStore(
		ownerID,
		"Test Store",
		"test-store",
		"A test store description",
		nil,
		string(valueobjects.PlanTypeBasic),
	)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return store
}

func newTestGRPCServer(
	repository *serverStoreRepository,
	capacity *serverProductCapacity,
	active *serverActiveStoreLookup,
) *storegrpc.Server {
	return storegrpc.NewServer(repository, capacity, active)
}

func TestGetStoreIDByOwnerID(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	store := newServerStore(t, ownerID)
	repository := &serverStoreRepository{ownerStore: store}
	server := newTestGRPCServer(repository, &serverProductCapacity{}, &serverActiveStoreLookup{})

	response, err := server.GetStoreIDByOwnerID(context.Background(), &storev1.GetStoreIDByOwnerIDRequest{
		OwnerId: ownerID.String(),
	})
	if err != nil {
		t.Fatalf("GetStoreIDByOwnerID: %v", err)
	}
	if response.GetStoreId() != store.ID().Value().String() {
		t.Fatalf("unexpected store ID: %s", response.GetStoreId())
	}
}

func TestGetStoreIDByOwnerIDMapsNotFound(t *testing.T) {
	t.Parallel()

	server := newTestGRPCServer(
		&serverStoreRepository{findByOwnerErr: applicationerrors.ErrStoreNotFound},
		&serverProductCapacity{},
		&serverActiveStoreLookup{},
	)

	_, err := server.GetStoreIDByOwnerID(context.Background(), &storev1.GetStoreIDByOwnerIDRequest{
		OwnerId: uuid.NewString(),
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestProductCountRPCs(t *testing.T) {
	t.Parallel()

	storeID := uuid.New()
	capacity := &serverProductCapacity{}
	server := newTestGRPCServer(&serverStoreRepository{}, capacity, &serverActiveStoreLookup{})

	_, err := server.IncrementProductCount(context.Background(), &storev1.IncrementProductCountRequest{
		StoreId: storeID.String(),
	})
	if err != nil {
		t.Fatalf("IncrementProductCount: %v", err)
	}
	if capacity.incremented != storeID {
		t.Fatalf("increment received store ID %s, want %s", capacity.incremented, storeID)
	}

	_, err = server.DecrementProductCount(context.Background(), &storev1.DecrementProductCountRequest{
		StoreId: storeID.String(),
	})
	if err != nil {
		t.Fatalf("DecrementProductCount: %v", err)
	}
	if capacity.decremented != storeID {
		t.Fatalf("decrement received store ID %s, want %s", capacity.decremented, storeID)
	}
}

func TestIncrementProductCountMapsDomainErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"unknown store", applicationerrors.ErrStoreNotFound, codes.NotFound},
		{"capacity exceeded", applicationerrors.ErrStoreProductCapacityExceeded, codes.ResourceExhausted},
		{"plan limit exceeded", applicationerrors.ErrStorePlanProductLimitExceeded, codes.ResourceExhausted},
		{"unexpected failure", errors.New("database unavailable"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := newTestGRPCServer(
				&serverStoreRepository{},
				&serverProductCapacity{incrementErr: tt.err},
				&serverActiveStoreLookup{},
			)

			_, err := server.IncrementProductCount(context.Background(), &storev1.IncrementProductCountRequest{
				StoreId: uuid.NewString(),
			})
			if status.Code(err) != tt.want {
				t.Fatalf("expected %s, got %s (%v)", tt.want, status.Code(err), err)
			}
		})
	}
}

func TestDecrementProductCountMapsUnderflow(t *testing.T) {
	t.Parallel()

	server := newTestGRPCServer(
		&serverStoreRepository{},
		&serverProductCapacity{decrementErr: applicationerrors.ErrStoreProductCountUnderflow},
		&serverActiveStoreLookup{},
	)

	_, err := server.DecrementProductCount(context.Background(), &storev1.DecrementProductCountRequest{
		StoreId: uuid.NewString(),
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func TestIsStoreActive(t *testing.T) {
	t.Parallel()

	store := newServerStore(t, uuid.New())
	server := newTestGRPCServer(
		&serverStoreRepository{store: store},
		&serverProductCapacity{},
		&serverActiveStoreLookup{},
	)

	response, err := server.IsStoreActive(context.Background(), &storev1.IsStoreActiveRequest{
		StoreId: store.ID().Value().String(),
	})
	if err != nil {
		t.Fatalf("IsStoreActive: %v", err)
	}
	if response.GetActive() != (store.Status() == valueobjects.StatusActive) {
		t.Fatalf("unexpected active result: %v", response.GetActive())
	}
}

func TestIsStoreActiveMapsNotFound(t *testing.T) {
	t.Parallel()

	server := newTestGRPCServer(
		&serverStoreRepository{findByIDErr: applicationerrors.ErrStoreNotFound},
		&serverProductCapacity{},
		&serverActiveStoreLookup{},
	)

	_, err := server.IsStoreActive(context.Background(), &storev1.IsStoreActiveRequest{
		StoreId: uuid.NewString(),
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestFilterActiveStoreIDs(t *testing.T) {
	t.Parallel()

	first, second := uuid.New(), uuid.New()
	active := &serverActiveStoreLookup{ids: []uuid.UUID{first, second}}
	server := newTestGRPCServer(&serverStoreRepository{}, &serverProductCapacity{}, active)

	response, err := server.FilterActiveStoreIDs(context.Background(), &storev1.FilterActiveStoreIDsRequest{
		StoreIds: []string{first.String(), second.String()},
	})
	if err != nil {
		t.Fatalf("FilterActiveStoreIDs: %v", err)
	}
	if len(response.GetActiveStoreIds()) != 2 {
		t.Fatalf("expected 2 active IDs, got %d", len(response.GetActiveStoreIds()))
	}
	if response.GetActiveStoreIds()[0] != first.String() || response.GetActiveStoreIds()[1] != second.String() {
		t.Fatalf("unexpected active IDs: %v", response.GetActiveStoreIds())
	}
}

func TestRPCsRejectInvalidUUIDs(t *testing.T) {
	t.Parallel()

	server := newTestGRPCServer(
		&serverStoreRepository{},
		&serverProductCapacity{},
		&serverActiveStoreLookup{},
	)

	tests := []struct {
		name string
		call func() error
	}{
		{"owner lookup", func() error {
			_, err := server.GetStoreIDByOwnerID(context.Background(), &storev1.GetStoreIDByOwnerIDRequest{OwnerId: "bad"})
			return err
		}},
		{"increment", func() error {
			_, err := server.IncrementProductCount(context.Background(), &storev1.IncrementProductCountRequest{StoreId: "bad"})
			return err
		}},
		{"decrement", func() error {
			_, err := server.DecrementProductCount(context.Background(), &storev1.DecrementProductCountRequest{StoreId: uuid.Nil.String()})
			return err
		}},
		{"active lookup", func() error {
			_, err := server.IsStoreActive(context.Background(), &storev1.IsStoreActiveRequest{StoreId: "bad"})
			return err
		}},
		{"active filter", func() error {
			_, err := server.FilterActiveStoreIDs(context.Background(), &storev1.FilterActiveStoreIDsRequest{StoreIds: []string{uuid.NewString(), "bad"}})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.call(); status.Code(err) != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", err)
			}
		})
	}
}
