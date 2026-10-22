package entities_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

func TestReconstituteCartPreservesState(t *testing.T) {
	cartID := uuid.New()
	customerID := uuid.New()
	firstItemID := uuid.New()
	secondItemID := uuid.New()
	firstProductID := uuid.New()
	secondProductID := uuid.New()
	firstStoreID := uuid.New()
	secondStoreID := uuid.New()
	createdAt := time.Date(2026, 10, 1, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 10, 2, 11, 45, 0, 0, time.UTC)

	cart, err := entities.ReconstituteCart(
		cartID,
		customerID,
		[]entities.CartItemState{
			{
				ID:        firstItemID,
				ProductID: firstProductID,
				StoreID:   firstStoreID,
				Quantity:  0,
			},
			{
				ID:        secondItemID,
				ProductID: secondProductID,
				StoreID:   secondStoreID,
				Quantity:  3,
			},
		},
		createdAt,
		updatedAt,
	)
	if err != nil {
		t.Fatalf("ReconstituteCart() error = %v", err)
	}

	if cart.ID() != cartID {
		t.Errorf("cart ID = %v, want %v", cart.ID(), cartID)
	}
	if cart.CustomerID() != customerID {
		t.Errorf("customer ID = %v, want %v", cart.CustomerID(), customerID)
	}
	if !cart.CreatedAt().Equal(createdAt) {
		t.Errorf("created at = %v, want %v", cart.CreatedAt(), createdAt)
	}
	if !cart.UpdatedAt().Equal(updatedAt) {
		t.Errorf("updated at = %v, want %v", cart.UpdatedAt(), updatedAt)
	}

	items := cart.Items()
	if len(items) != 2 {
		t.Fatalf("item count = %d, want 2", len(items))
	}

	if items[0].ID() != firstItemID ||
		items[0].ProductID() != firstProductID ||
		items[0].StoreID() != firstStoreID ||
		items[0].Quantity() != 0 {
		t.Errorf("first item state was not preserved: %+v", items[0])
	}

	if items[1].ID() != secondItemID ||
		items[1].ProductID() != secondProductID ||
		items[1].StoreID() != secondStoreID ||
		items[1].Quantity() != 3 {
		t.Errorf("second item state was not preserved: %+v", items[1])
	}
}

func TestReconstituteCartRejectsInvalidState(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name       string
		cartID     uuid.UUID
		customerID uuid.UUID
		items      []entities.CartItemState
		wantErr    error
	}{
		{
			name:       "nil cart ID",
			cartID:     uuid.Nil,
			customerID: uuid.New(),
			wantErr:    domainerrors.ErrInvalidCartID,
		},
		{
			name:       "nil customer ID",
			cartID:     uuid.New(),
			customerID: uuid.Nil,
			wantErr:    domainerrors.ErrInvalidCustomerID,
		},
		{
			name:       "nil product ID",
			cartID:     uuid.New(),
			customerID: uuid.New(),
			items: []entities.CartItemState{{
				ID:       uuid.New(),
				StoreID:  uuid.New(),
				Quantity: 1,
			}},
			wantErr: domainerrors.ErrInvalidProductID,
		},
		{
			name:       "negative quantity",
			cartID:     uuid.New(),
			customerID: uuid.New(),
			items: []entities.CartItemState{{
				ID:        uuid.New(),
				ProductID: uuid.New(),
				StoreID:   uuid.New(),
				Quantity:  -1,
			}},
			wantErr: domainerrors.ErrInvalidQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entities.ReconstituteCart(
				tt.cartID,
				tt.customerID,
				tt.items,
				now,
				now,
			)
			if err != tt.wantErr {
				t.Errorf("ReconstituteCart() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
