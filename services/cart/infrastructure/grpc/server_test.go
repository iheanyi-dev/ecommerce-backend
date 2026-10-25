package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	cartv1 "github.com/iheanyi-dev/ecommerce-backend/proto/cart/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type checkoutMock struct {
	result        *use_cases.CheckoutCartOutput
	err           error
	called        bool
	gotCustomerID uuid.UUID
	gotCartID     uuid.UUID
}

func (m *checkoutMock) Execute(
	_ context.Context,
	customerID, cartID uuid.UUID,
) (*use_cases.CheckoutCartOutput, error) {
	m.called = true
	m.gotCustomerID = customerID
	m.gotCartID = cartID
	return m.result, m.err
}

func TestCheckoutCartForOrderRejectsNilRequest(t *testing.T) {
	server := NewServer(&use_cases.CheckoutCartForOrder{})

	_, err := server.CheckoutCartForOrder(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestCheckoutCartForOrderRejectsInvalidUUIDs(t *testing.T) {
	server := NewServer(&use_cases.CheckoutCartForOrder{})

	tests := []struct {
		name string
		req  *cartv1.CheckoutCartForOrderRequest
	}{
		{
			name: "invalid customer ID",
			req: &cartv1.CheckoutCartForOrderRequest{
				CustomerId: "not-a-uuid",
				CartId:     uuid.NewString(),
			},
		},
		{
			name: "nil customer UUID",
			req: &cartv1.CheckoutCartForOrderRequest{
				CustomerId: uuid.Nil.String(),
				CartId:     uuid.NewString(),
			},
		},
		{
			name: "invalid cart ID",
			req: &cartv1.CheckoutCartForOrderRequest{
				CustomerId: uuid.NewString(),
				CartId:     "not-a-uuid",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.CheckoutCartForOrder(context.Background(), tt.req)
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("expected InvalidArgument, got %v", err)
			}
		})
	}
}

func TestCheckoutCartForOrderMapsSuccessfulResult(t *testing.T) {
	customerID, cartID, storeID, productID, removedID :=
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()

	mock := &checkoutMock{
		result: &use_cases.CheckoutCartOutput{
			CartID: cartID,
			Stores: []use_cases.CheckoutStore{
				{
					StoreID: storeID,
					Items: []use_cases.CheckoutItem{
						{
							Product: ports.ProductSnapshot{
								ID:         productID,
								StoreID:    storeID,
								Name:       "Notebook",
								PriceMinor: 12500,
								Currency:   "NGN",
								Quantity:   7,
							},
							CartQuantity: 2,
						},
					},
				},
			},
			RemovedProductIDs: []uuid.UUID{removedID},
		},
	}
	server := &Server{checkout: mock}

	response, err := server.CheckoutCartForOrder(context.Background(), &cartv1.CheckoutCartForOrderRequest{
		CustomerId: customerID.String(),
		CartId:     cartID.String(),
	})
	if err != nil {
		t.Fatalf("CheckoutCartForOrder returned error: %v", err)
	}
	if !mock.called || mock.gotCustomerID != customerID || mock.gotCartID != cartID {
		t.Fatal("checkout use case was not called with the requested identities")
	}
	if response.GetCartId() != cartID.String() {
		t.Fatalf("unexpected cart ID: %s", response.GetCartId())
	}
	if len(response.GetStores()) != 1 || response.GetStores()[0].GetStoreId() != storeID.String() {
		t.Fatalf("unexpected store mapping: %+v", response.GetStores())
	}
	if len(response.GetStores()[0].GetItems()) != 1 {
		t.Fatalf("unexpected item count: %d", len(response.GetStores()[0].GetItems()))
	}
	item := response.GetStores()[0].GetItems()[0]
	if item.GetProductId() != productID.String() ||
		item.GetName() != "Notebook" ||
		item.GetQuantity() != 2 ||
		item.GetPriceMinor() != 12500 ||
		item.GetCurrency() != "NGN" {
		t.Fatalf("unexpected item mapping: %+v", item)
	}
	if len(response.GetRemovedProductIds()) != 1 || response.GetRemovedProductIds()[0] != removedID.String() {
		t.Fatalf("unexpected removed product IDs: %+v", response.GetRemovedProductIds())
	}
}

func TestCheckoutCartForOrderMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"cart missing", domainerrors.ErrCartNotFound, codes.NotFound},
		{"cart ownership mismatch", domainerrors.ErrCartOwnershipMismatch, codes.NotFound},
		{"invalid customer", domainerrors.ErrInvalidCustomerID, codes.InvalidArgument},
		{"unexpected error", errors.New("database unavailable"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &checkoutMock{err: tt.err}
			server := &Server{checkout: mock}
			_, err := server.CheckoutCartForOrder(context.Background(), &cartv1.CheckoutCartForOrderRequest{
				CustomerId: uuid.NewString(),
				CartId:     uuid.NewString(),
			})
			if status.Code(err) != tt.want {
				t.Fatalf("expected %s, got %v", tt.want, err)
			}
		})
	}
}
