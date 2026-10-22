package product_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	productv1 "github.com/iheanyi-dev/ecommerce-backend/proto/product/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type productServiceStub struct {
	response *productv1.GetProductsByIDsResponse
	err      error
	request  *productv1.GetProductsByIDsRequest
	md       metadata.MD
	calls    int
}

func (s *productServiceStub) SearchProducts(
	context.Context,
	*productv1.SearchProductsRequest,
	...grpc.CallOption,
) (*productv1.SearchProductsResponse, error) {
	return nil, errors.New("unexpected SearchProducts call")
}

func (s *productServiceStub) CountProductsByStore(
	context.Context,
	*productv1.CountProductsByStoreRequest,
	...grpc.CallOption,
) (*productv1.CountProductsByStoreResponse, error) {
	return nil, errors.New("unexpected CountProductsByStore call")
}

func (s *productServiceStub) GetProductsByIDs(
	ctx context.Context,
	req *productv1.GetProductsByIDsRequest,
	_ ...grpc.CallOption,
) (*productv1.GetProductsByIDsResponse, error) {
	s.calls++
	s.request = req
	s.md, _ = metadata.FromOutgoingContext(ctx)
	return s.response, s.err
}

func TestGetProductsByIDsMapsSnapshotsAndAuthentication(t *testing.T) {
	productID := uuid.New()
	storeID := uuid.New()
	missingID := uuid.New()

	stub := &productServiceStub{
		response: &productv1.GetProductsByIDsResponse{
			Products: []*productv1.ProductSnapshot{
				{
					ProductId:   productID.String(),
					StoreId:     storeID.String(),
					Name:        "Keyboard",
					Description: "Mechanical keyboard",
					PriceMinor:  125000,
					Currency:    "NGN",
					Quantity:    7,
				},
			},
		},
	}

	client := product.NewClientWithStub(stub, "cart", "test-secret")
	got, err := client.GetProductsByIDs(
		context.Background(),
		[]uuid.UUID{productID, missingID},
	)
	if err != nil {
		t.Fatalf("GetProductsByIDs() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d snapshots, want 1; missing IDs should be omitted", len(got))
	}

	want := struct {
		ID          uuid.UUID
		StoreID     uuid.UUID
		Name        string
		Description string
		PriceMinor  int64
		Currency    string
		Quantity    int64
	}{
		ID:          productID,
		StoreID:     storeID,
		Name:        "Keyboard",
		Description: "Mechanical keyboard",
		PriceMinor:  125000,
		Currency:    "NGN",
		Quantity:    7,
	}

	if got[0].ID != want.ID ||
		got[0].StoreID != want.StoreID ||
		got[0].Name != want.Name ||
		got[0].Description != want.Description ||
		got[0].PriceMinor != want.PriceMinor ||
		got[0].Currency != want.Currency ||
		got[0].Quantity != want.Quantity {
		t.Fatalf("snapshot = %+v, want %+v", got[0], want)
	}

	if !reflect.DeepEqual(stub.request.GetProductIds(), []string{
		productID.String(),
		missingID.String(),
	}) {
		t.Fatalf("request IDs = %v", stub.request.GetProductIds())
	}

	if got := stub.md.Get("x-service-name"); !reflect.DeepEqual(got, []string{"cart"}) {
		t.Errorf("service name metadata = %v", got)
	}
	if got := stub.md.Get("x-service-secret"); !reflect.DeepEqual(got, []string{"test-secret"}) {
		t.Errorf("service secret metadata = %v", got)
	}
}

func TestGetProductsByIDsReturnsRPCError(t *testing.T) {
	stub := &productServiceStub{err: errors.New("Product service unavailable")}
	client := product.NewClientWithStub(stub, "cart", "test-secret")

	_, err := client.GetProductsByIDs(context.Background(), []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected RPC error")
	}
	if stub.calls != 1 {
		t.Fatalf("RPC calls = %d, want 1", stub.calls)
	}
}

func TestGetProductsByIDsEmptyInputSkipsRPC(t *testing.T) {
	stub := &productServiceStub{}
	client := product.NewClientWithStub(stub, "cart", "test-secret")

	got, err := client.GetProductsByIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetProductsByIDs() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("result = %#v, want non-nil empty slice", got)
	}
	if stub.calls != 0 {
		t.Fatalf("RPC calls = %d, want 0", stub.calls)
	}
}

func TestGetProductsByIDsRejectsInvalidProductID(t *testing.T) {
	stub := &productServiceStub{
		response: &productv1.GetProductsByIDsResponse{
			Products: []*productv1.ProductSnapshot{
				{ProductId: "not-a-uuid", StoreId: uuid.NewString()},
			},
		},
	}
	client := product.NewClientWithStub(stub, "cart", "test-secret")

	_, err := client.GetProductsByIDs(context.Background(), []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected invalid product ID error")
	}
}

func TestGetProductsByIDsRejectsInvalidStoreID(t *testing.T) {
	stub := &productServiceStub{
		response: &productv1.GetProductsByIDsResponse{
			Products: []*productv1.ProductSnapshot{
				{ProductId: uuid.NewString(), StoreId: "not-a-uuid"},
			},
		},
	}
	client := product.NewClientWithStub(stub, "cart", "test-secret")

	_, err := client.GetProductsByIDs(context.Background(), []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected invalid Store ID error")
	}
}

func TestGetProductsByIDsRejectsNilResponse(t *testing.T) {
	stub := &productServiceStub{}
	client := product.NewClientWithStub(stub, "cart", "test-secret")

	_, err := client.GetProductsByIDs(context.Background(), []uuid.UUID{uuid.New()})
	if err == nil {
		t.Fatal("expected empty response error")
	}
}
