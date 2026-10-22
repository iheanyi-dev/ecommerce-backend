package redis_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
	redisrepo "github.com/iheanyi-dev/ecommerce-backend/services/cart/infrastructure/persistence/redis"
)

func newTestRepository(t *testing.T) *redisrepo.CartRepository {
	t.Helper()

	server := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{
		Addr: server.Addr(),
	})

	t.Cleanup(func() {
		_ = client.Close()
	})

	return redisrepo.NewCartRepository(client)
}

func TestSaveAndGetByCustomerID(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)

	customerID := uuid.New()
	productID := uuid.New()
	storeID := uuid.New()

	cart, err := entities.NewCart(customerID)
	if err != nil {
		t.Fatalf("NewCart() error = %v", err)
	}
	if err := cart.AddItem(productID, storeID, 3); err != nil {
		t.Fatalf("AddItem() error = %v", err)
	}

	if err := repo.Save(ctx, cart); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.GetByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("GetByCustomerID() error = %v", err)
	}

	if got.ID() != cart.ID() {
		t.Errorf("cart ID = %v, want %v", got.ID(), cart.ID())
	}
	if got.CustomerID() != customerID {
		t.Errorf("customer ID = %v, want %v", got.CustomerID(), customerID)
	}
	if !got.CreatedAt().Equal(cart.CreatedAt()) {
		t.Errorf("created timestamp = %v, want %v", got.CreatedAt(), cart.CreatedAt())
	}
	if !got.UpdatedAt().Equal(cart.UpdatedAt()) {
		t.Errorf("updated timestamp = %v, want %v", got.UpdatedAt(), cart.UpdatedAt())
	}

	items := got.Items()
	if len(items) != 1 {
		t.Fatalf("item count = %d, want 1", len(items))
	}
	if items[0].ID() != cart.Items()[0].ID() {
		t.Errorf("item ID = %v, want %v", items[0].ID(), cart.Items()[0].ID())
	}
	if items[0].ProductID() != productID {
		t.Errorf("product ID = %v, want %v", items[0].ProductID(), productID)
	}
	if items[0].StoreID() != storeID {
		t.Errorf("store ID = %v, want %v", items[0].StoreID(), storeID)
	}
	if items[0].Quantity() != 3 {
		t.Errorf("quantity = %d, want 3", items[0].Quantity())
	}
}

func TestGetByCustomerIDReturnsNotFoundForMissingCart(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.GetByCustomerID(context.Background(), uuid.New())
	if !errors.Is(err, domainerrors.ErrCartNotFound) {
		t.Fatalf("error = %v, want ErrCartNotFound", err)
	}
}

func TestDeleteByCustomerID(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)

	customerID := uuid.New()
	cart, err := entities.NewCart(customerID)
	if err != nil {
		t.Fatalf("NewCart() error = %v", err)
	}
	if err := repo.Save(ctx, cart); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := repo.DeleteByCustomerID(ctx, customerID); err != nil {
		t.Fatalf("DeleteByCustomerID() error = %v", err)
	}

	_, err = repo.GetByCustomerID(ctx, customerID)
	if !errors.Is(err, domainerrors.ErrCartNotFound) {
		t.Fatalf("error after deletion = %v, want ErrCartNotFound", err)
	}
}

func TestDeleteMissingCartIsHarmless(t *testing.T) {
	repo := newTestRepository(t)

	if err := repo.DeleteByCustomerID(context.Background(), uuid.New()); err != nil {
		t.Fatalf("deleting a missing cart returned error: %v", err)
	}
}

func TestSaveNilCartReturnsError(t *testing.T) {
	repo := newTestRepository(t)

	if err := repo.Save(context.Background(), nil); err == nil {
		t.Fatal("Save(nil) expected an error, got nil")
	}
}

func TestSaveAndGetPreservesItemOrder(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)

	customerID := uuid.New()
	storeID := uuid.New()
	firstProductID := uuid.New()
	secondProductID := uuid.New()

	cart, err := entities.NewCart(customerID)
	if err != nil {
		t.Fatalf("NewCart() error = %v", err)
	}
	if err := cart.AddItem(firstProductID, storeID, 1); err != nil {
		t.Fatalf("AddItem(first) error = %v", err)
	}
	if err := cart.AddItem(secondProductID, storeID, 2); err != nil {
		t.Fatalf("AddItem(second) error = %v", err)
	}

	if err := repo.Save(ctx, cart); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := repo.GetByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("GetByCustomerID() error = %v", err)
	}

	items := got.Items()
	if len(items) != 2 {
		t.Fatalf("item count = %d, want 2", len(items))
	}
	if items[0].ProductID() != firstProductID {
		t.Errorf("first product ID = %v, want %v", items[0].ProductID(), firstProductID)
	}
	if items[1].ProductID() != secondProductID {
		t.Errorf("second product ID = %v, want %v", items[1].ProductID(), secondProductID)
	}
}
