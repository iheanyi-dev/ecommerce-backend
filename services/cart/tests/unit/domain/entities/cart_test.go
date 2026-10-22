package entities_test

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

func TestNewCart(t *testing.T) {
	customerID := uuid.New()

	cart, err := entities.NewCart(customerID)
	if err != nil {
		t.Fatalf("NewCart() error = %v", err)
	}

	if cart.ID() == uuid.Nil {
		t.Fatal("NewCart() generated a nil cart ID")
	}
	if cart.CustomerID() != customerID {
		t.Fatalf("CustomerID() = %v, want %v", cart.CustomerID(), customerID)
	}
	if cart.Items() == nil {
		t.Fatal("Items() should return an initialized empty slice")
	}
	if len(cart.Items()) != 0 {
		t.Fatalf("new cart has %d items, want 0", len(cart.Items()))
	}
	if cart.CreatedAt().IsZero() || cart.UpdatedAt().IsZero() {
		t.Fatal("cart timestamps should be initialized")
	}
	if cart.UpdatedAt().Before(cart.CreatedAt()) {
		t.Fatal("UpdatedAt should not precede CreatedAt")
	}
}

func TestNewCartRejectsNilCustomerID(t *testing.T) {
	_, err := entities.NewCart(uuid.Nil)
	if !errors.Is(err, domainerrors.ErrInvalidCustomerID) {
		t.Fatalf("NewCart() error = %v, want ErrInvalidCustomerID", err)
	}
}

func TestAddItem(t *testing.T) {
	cart, err := entities.NewCart(uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	productID := uuid.New()
	storeID := uuid.New()

	if err := cart.AddItem(productID, storeID, 2); err != nil {
		t.Fatalf("AddItem() error = %v", err)
	}

	items := cart.Items()
	if len(items) != 1 {
		t.Fatalf("Items() length = %d, want 1", len(items))
	}

	item := items[0]
	if item.ID() == uuid.Nil {
		t.Fatal("item ID should be a UUID")
	}
	if item.ProductID() != productID {
		t.Fatalf("ProductID() = %v, want %v", item.ProductID(), productID)
	}
	if item.StoreID() != storeID {
		t.Fatalf("StoreID() = %v, want %v", item.StoreID(), storeID)
	}
	if item.Quantity() != 2 {
		t.Fatalf("Quantity() = %d, want 2", item.Quantity())
	}
}

func TestAddItemSetsQuantityForExistingProduct(t *testing.T) {
	cart, err := entities.NewCart(uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	productID := uuid.New()
	storeID := uuid.New()

	if err := cart.AddItem(productID, storeID, 2); err != nil {
		t.Fatal(err)
	}
	firstID := cart.Items()[0].ID()

	if err := cart.AddItem(productID, storeID, 3); err != nil {
		t.Fatal(err)
	}

	items := cart.Items()
	if len(items) != 1 {
		t.Fatalf("Items() length = %d, want 1", len(items))
	}
	if items[0].Quantity() != 3 {
		t.Fatalf("Quantity() = %d, want 3", items[0].Quantity())
	}
	if items[0].ID() != firstID {
		t.Fatal("updating an item should preserve its ID")
	}
}

func TestCartSupportsProductsFromMultipleStores(t *testing.T) {
	cart, err := entities.NewCart(uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	if err := cart.AddItem(uuid.New(), uuid.New(), 1); err != nil {
		t.Fatal(err)
	}
	if err := cart.AddItem(uuid.New(), uuid.New(), 2); err != nil {
		t.Fatal(err)
	}

	if got := len(cart.Items()); got != 2 {
		t.Fatalf("Items() length = %d, want 2", got)
	}
}

func TestAddItemRejectsNilProductID(t *testing.T) {
	cart, _ := entities.NewCart(uuid.New())

	err := cart.AddItem(uuid.Nil, uuid.New(), 1)
	if !errors.Is(err, domainerrors.ErrInvalidProductID) {
		t.Fatalf("AddItem() error = %v, want ErrInvalidProductID", err)
	}
}

func TestAddItemRejectsNilStoreID(t *testing.T) {
	cart, _ := entities.NewCart(uuid.New())

	err := cart.AddItem(uuid.New(), uuid.Nil, 1)
	if !errors.Is(err, domainerrors.ErrInvalidStoreID) {
		t.Fatalf("AddItem() error = %v, want ErrInvalidStoreID", err)
	}
}

func TestAddItemRejectsNonPositiveQuantity(t *testing.T) {
	for _, quantity := range []int64{0, -1} {
		t.Run(string(rune('0'+quantity+1)), func(t *testing.T) {
			cart, _ := entities.NewCart(uuid.New())

			err := cart.AddItem(uuid.New(), uuid.New(), quantity)
			if !errors.Is(err, domainerrors.ErrInvalidQuantity) {
				t.Fatalf("AddItem() error = %v, want ErrInvalidQuantity", err)
			}
		})
	}
}

func TestAddItemRejectsProductWithConflictingStore(t *testing.T) {
	cart, _ := entities.NewCart(uuid.New())
	productID := uuid.New()

	if err := cart.AddItem(productID, uuid.New(), 1); err != nil {
		t.Fatal(err)
	}

	err := cart.AddItem(productID, uuid.New(), 1)
	if !errors.Is(err, domainerrors.ErrProductStoreMismatch) {
		t.Fatalf("AddItem() error = %v, want ErrProductStoreMismatch", err)
	}
}

func TestAddItemCanOverwriteLargeQuantity(t *testing.T) {
	cart, _ := entities.NewCart(uuid.New())
	productID := uuid.New()
	storeID := uuid.New()

	if err := cart.AddItem(productID, storeID, math.MaxInt64); err != nil {
		t.Fatalf("AddItem(MaxInt64) error = %v", err)
	}

	if err := cart.AddItem(productID, storeID, 1); err != nil {
		t.Fatalf("AddItem(1) error = %v", err)
	}

	if got := cart.Items()[0].Quantity(); got != 1 {
		t.Fatalf("quantity = %d, want 1", got)
	}
}
