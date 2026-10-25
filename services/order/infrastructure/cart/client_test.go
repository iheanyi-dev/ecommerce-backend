package cart

import (
	"testing"

	"github.com/google/uuid"
	cartv1 "github.com/iheanyi-dev/ecommerce-backend/proto/cart/v1"
)

func TestMapCheckoutResponse(t *testing.T) {
	cartID, storeID, productID, removedID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	response := &cartv1.CheckoutCartForOrderResponse{
		CartId: cartID.String(),
		Stores: []*cartv1.CheckoutStore{
			{
				StoreId: storeID.String(),
				Items: []*cartv1.CheckoutItem{
					{
						ProductId:  productID.String(),
						Name:       "Notebook",
						Quantity:   2,
						PriceMinor: 12500,
						Currency:   "NGN",
					},
				},
			},
		},
		RemovedProductIds: []string{removedID.String()},
	}

	output, err := mapCheckoutResponse(response, cartID)
	if err != nil {
		t.Fatalf("mapCheckoutResponse returned error: %v", err)
	}
	if output.CartID != cartID {
		t.Fatalf("unexpected cart ID: %s", output.CartID)
	}
	if len(output.Stores) != 1 || output.Stores[0].StoreID != storeID {
		t.Fatalf("unexpected store mapping: %+v", output.Stores)
	}
	if len(output.Stores[0].Items) != 1 {
		t.Fatalf("unexpected item count: %d", len(output.Stores[0].Items))
	}
	item := output.Stores[0].Items[0]
	if item.ProductID != productID || item.Name != "Notebook" ||
		item.Quantity != 2 || item.PriceMinor != 12500 || item.Currency != "NGN" {
		t.Fatalf("unexpected item mapping: %+v", item)
	}
	if len(output.RemovedProductIDs) != 1 || output.RemovedProductIDs[0] != removedID {
		t.Fatalf("unexpected removed IDs: %+v", output.RemovedProductIDs)
	}
}

func TestMapCheckoutResponseRejectsMalformedData(t *testing.T) {
	cartID := uuid.New()

	tests := []struct {
		name     string
		response *cartv1.CheckoutCartForOrderResponse
	}{
		{"nil response", nil},
		{"invalid cart ID", &cartv1.CheckoutCartForOrderResponse{CartId: "bad-id"}},
		{"different cart ID", &cartv1.CheckoutCartForOrderResponse{CartId: uuid.NewString()}},
		{"nil store", &cartv1.CheckoutCartForOrderResponse{CartId: cartID.String(), Stores: []*cartv1.CheckoutStore{nil}}},
		{
			"invalid store ID",
			&cartv1.CheckoutCartForOrderResponse{
				CartId: cartID.String(),
				Stores: []*cartv1.CheckoutStore{{StoreId: "bad-id"}},
			},
		},
		{
			"invalid product ID",
			&cartv1.CheckoutCartForOrderResponse{
				CartId: cartID.String(),
				Stores: []*cartv1.CheckoutStore{{
					StoreId: uuid.NewString(),
					Items: []*cartv1.CheckoutItem{{
						ProductId: "bad-id", Quantity: 1, PriceMinor: 100, Currency: "NGN",
					}},
				}},
			},
		},
		{
			"nonpositive quantity",
			&cartv1.CheckoutCartForOrderResponse{
				CartId: cartID.String(),
				Stores: []*cartv1.CheckoutStore{{
					StoreId: uuid.NewString(),
					Items: []*cartv1.CheckoutItem{{
						ProductId: uuid.NewString(), Quantity: 0, PriceMinor: 100, Currency: "NGN",
					}},
				}},
			},
		},
		{
			"negative price",
			&cartv1.CheckoutCartForOrderResponse{
				CartId: cartID.String(),
				Stores: []*cartv1.CheckoutStore{{
					StoreId: uuid.NewString(),
					Items: []*cartv1.CheckoutItem{{
						ProductId: uuid.NewString(), Quantity: 1, PriceMinor: -1, Currency: "NGN",
					}},
				}},
			},
		},
		{
			"invalid removed product ID",
			&cartv1.CheckoutCartForOrderResponse{
				CartId:            cartID.String(),
				RemovedProductIds: []string{"bad-id"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := mapCheckoutResponse(tt.response, cartID); err == nil {
				t.Fatal("expected malformed response to be rejected")
			}
		})
	}
}
