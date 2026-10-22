package use_cases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
)

type checkoutProductClientMock struct {
	snapshots []ports.ProductSnapshot
	err       error
	calls     int
}

func (m *checkoutProductClientMock) GetProductsByIDs(
	_ context.Context,
	_ []uuid.UUID,
) ([]ports.ProductSnapshot, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.snapshots, nil
}

func newCheckoutCart(t *testing.T, quantities ...int64) (*entities.Cart, []uuid.UUID, []uuid.UUID) {
	t.Helper()

	cart, err := entities.NewCart(uuid.New())
	if err != nil {
		t.Fatal(err)
	}

	productIDs := make([]uuid.UUID, 0, len(quantities))
	storeIDs := make([]uuid.UUID, 0, len(quantities))

	for _, quantity := range quantities {
		productID := uuid.New()
		storeID := uuid.New()

		if err := cart.AddItem(productID, storeID, quantity); err != nil {
			t.Fatal(err)
		}

		productIDs = append(productIDs, productID)
		storeIDs = append(storeIDs, storeID)
	}

	return cart, productIDs, storeIDs
}

func TestCheckoutCartRefreshesProductsClampsQuantitiesAndGroupsByStore(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, storeIDs := newCheckoutCart(t, 5, 2)

	repo := &cartRepositoryMock{cart: cart}
	logger := &loggerMock{}
	metrics := &metricsMock{}
	products := &checkoutProductClientMock{
		snapshots: []ports.ProductSnapshot{
			{
				ID:          productIDs[0],
				StoreID:     storeIDs[0],
				Name:        "Updated Product",
				Description: "Current description",
				PriceMinor:  25000,
				Currency:    "NGN",
				Quantity:    3,
			},
			{
				ID:          productIDs[1],
				StoreID:     storeIDs[1],
				Name:        "Second Product",
				Description: "Another product",
				PriceMinor:  12000,
				Currency:    "NGN",
				Quantity:    10,
			},
		},
	}

	uc := use_cases.NewCheckoutCart(repo, products, logger, metrics)
	result, err := uc.Execute(ctx, cart.CustomerID())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Stores) != 2 {
		t.Fatalf("expected 2 store groups, got %d", len(result.Stores))
	}
	if result.Stores[0].StoreID != storeIDs[0] ||
		result.Stores[1].StoreID != storeIDs[1] {
		t.Fatal("checkout items were not grouped in cart encounter order by store")
	}

	first := result.Stores[0].Items[0]
	if first.CartQuantity != 3 {
		t.Fatalf("expected cart quantity to be clamped to 3, got %d", first.CartQuantity)
	}
	if first.Product.PriceMinor != 25000 ||
		first.Product.Name != "Updated Product" ||
		first.Product.Description != "Current description" {
		t.Fatal("checkout did not return refreshed product details")
	}
	if first.Product.Quantity != 3 {
		t.Fatalf("expected authoritative stock quantity 3, got %d", first.Product.Quantity)
	}

	second := result.Stores[1].Items[0]
	if second.CartQuantity != 2 {
		t.Fatalf("expected unchanged cart quantity 2, got %d", second.CartQuantity)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("expected one save after reconciliation, got %d", repo.saveCalls)
	}
	if cart.Items()[0].Quantity() != 3 {
		t.Fatal("reconciled quantity was not applied to the cart")
	}
}

func TestCheckoutCartAllowsZeroStockQuantity(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, storeIDs := newCheckoutCart(t, 4)

	repo := &cartRepositoryMock{cart: cart}
	products := &checkoutProductClientMock{
		snapshots: []ports.ProductSnapshot{
			{
				ID:       productIDs[0],
				StoreID:  storeIDs[0],
				Name:     "Out of Stock",
				Currency: "NGN",
				Quantity: 0,
			},
		},
	}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	result, err := uc.Execute(ctx, cart.CustomerID())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := result.Stores[0].Items[0].CartQuantity; got != 0 {
		t.Fatalf("expected cart quantity 0, got %d", got)
	}
	if got := cart.Items()[0].Quantity(); got != 0 {
		t.Fatalf("expected persisted cart quantity 0, got %d", got)
	}
	if err := cart.Validate(); err != nil {
		t.Fatalf("cart with a checkout-reconciled zero quantity should be valid: %v", err)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("expected one save, got %d", repo.saveCalls)
	}
}

func TestCheckoutCartRemovesMissingProductsAndReportsTheirIDs(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, _ := newCheckoutCart(t, 2)

	repo := &cartRepositoryMock{cart: cart}
	products := &checkoutProductClientMock{
		// A successful response that omits a requested ID means that
		// the Product service no longer has that product.
		snapshots: []ports.ProductSnapshot{},
	}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	result, err := uc.Execute(ctx, cart.CustomerID())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.RemovedProductIDs) != 1 ||
		result.RemovedProductIDs[0] != productIDs[0] {
		t.Fatalf("expected missing product ID to be reported, got %v", result.RemovedProductIDs)
	}
	if len(result.Stores) != 0 {
		t.Fatalf("expected no store groups, got %d", len(result.Stores))
	}
	if len(cart.Items()) != 0 {
		t.Fatal("missing product was not removed from the cart")
	}
	if repo.saveCalls != 1 {
		t.Fatalf("expected one save after removal, got %d", repo.saveCalls)
	}
}

func TestCheckoutCartDoesNotMutateCartWhenProductServiceFails(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, storeIDs := newCheckoutCart(t, 4)

	repo := &cartRepositoryMock{cart: cart}
	serviceErr := errors.New("product service unavailable")
	products := &checkoutProductClientMock{err: serviceErr}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	_, err := uc.Execute(ctx, cart.CustomerID())
	if err == nil {
		t.Fatal("expected Product service error")
	}

	if cart.Items()[0].Quantity() != 4 {
		t.Fatal("cart quantity changed after Product service failure")
	}
	if cart.Items()[0].ProductID() != productIDs[0] ||
		cart.Items()[0].StoreID() != storeIDs[0] {
		t.Fatal("cart item changed after Product service failure")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no save after Product service failure, got %d", repo.saveCalls)
	}
}

func TestCheckoutCartRejectsInvalidProductResponseWithoutMutation(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, storeIDs := newCheckoutCart(t, 4)

	repo := &cartRepositoryMock{cart: cart}
	products := &checkoutProductClientMock{
		snapshots: []ports.ProductSnapshot{
			{
				ID:       uuid.New(), // Not one of the requested products.
				StoreID:  uuid.New(),
				Currency: "NGN",
				Quantity: 3,
			},
		},
	}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	_, err := uc.Execute(ctx, cart.CustomerID())
	if err == nil {
		t.Fatal("expected invalid Product response error")
	}

	if cart.Items()[0].Quantity() != 4 ||
		cart.Items()[0].ProductID() != productIDs[0] ||
		cart.Items()[0].StoreID() != storeIDs[0] {
		t.Fatal("cart changed after receiving an invalid Product response")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no save after invalid Product response, got %d", repo.saveCalls)
	}
}

func TestCheckoutCartReturnsErrorWhenSavingReconciliationFails(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, storeIDs := newCheckoutCart(t, 5)

	repo := &cartRepositoryMock{
		cart:    cart,
		saveErr: errors.New("redis unavailable"),
	}
	products := &checkoutProductClientMock{
		snapshots: []ports.ProductSnapshot{
			{
				ID:       productIDs[0],
				StoreID:  storeIDs[0],
				Name:     "Product",
				Currency: "NGN",
				Quantity: 2,
			},
		},
	}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	result, err := uc.Execute(ctx, cart.CustomerID())

	if err == nil {
		t.Fatal("expected an error when saving the reconciled cart fails")
	}
	if result != nil {
		t.Fatal("expected no checkout result when reconciliation cannot be persisted")
	}
	if repo.saveCalls != 1 {
		t.Fatalf("expected one save attempt, got %d", repo.saveCalls)
	}
}

func TestCheckoutCartRejectsDuplicateProductSnapshotsWithoutMutation(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, storeIDs := newCheckoutCart(t, 4)

	snapshot := ports.ProductSnapshot{
		ID:       productIDs[0],
		StoreID:  storeIDs[0],
		Name:     "Product",
		Currency: "NGN",
		Quantity: 2,
	}

	repo := &cartRepositoryMock{cart: cart}
	products := &checkoutProductClientMock{
		snapshots: []ports.ProductSnapshot{snapshot, snapshot},
	}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	_, err := uc.Execute(ctx, cart.CustomerID())

	if err == nil {
		t.Fatal("expected duplicate Product snapshots to be rejected")
	}
	if cart.Items()[0].Quantity() != 4 ||
		cart.Items()[0].StoreID() != storeIDs[0] {
		t.Fatal("cart changed after duplicate Product snapshots")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no save after duplicate snapshots, got %d", repo.saveCalls)
	}
}

func TestCheckoutCartRejectsNegativeStockWithoutMutation(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, storeIDs := newCheckoutCart(t, 4)

	repo := &cartRepositoryMock{cart: cart}
	products := &checkoutProductClientMock{
		snapshots: []ports.ProductSnapshot{
			{
				ID:       productIDs[0],
				StoreID:  storeIDs[0],
				Currency: "NGN",
				Quantity: -1,
			},
		},
	}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	_, err := uc.Execute(ctx, cart.CustomerID())

	if err == nil {
		t.Fatal("expected negative stock quantity to be rejected")
	}
	if cart.Items()[0].Quantity() != 4 ||
		cart.Items()[0].StoreID() != storeIDs[0] {
		t.Fatal("cart changed after receiving a negative stock quantity")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no save after invalid stock quantity, got %d", repo.saveCalls)
	}
}

func TestCheckoutCartMovesProductToItsCurrentStore(t *testing.T) {
	ctx := context.Background()
	cart, productIDs, originalStoreIDs := newCheckoutCart(t, 3)
	currentStoreID := uuid.New()

	repo := &cartRepositoryMock{cart: cart}
	products := &checkoutProductClientMock{
		snapshots: []ports.ProductSnapshot{
			{
				ID:       productIDs[0],
				StoreID:  currentStoreID,
				Name:     "Moved Product",
				Currency: "NGN",
				Quantity: 8,
			},
		},
	}

	uc := use_cases.NewCheckoutCart(repo, products, &loggerMock{}, &metricsMock{})
	result, err := uc.Execute(ctx, cart.CustomerID())

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Stores) != 1 || result.Stores[0].StoreID != currentStoreID {
		t.Fatalf("expected checkout grouped under current store %s, got %+v",
			currentStoreID, result.Stores)
	}
	if got := cart.Items()[0].StoreID(); got != currentStoreID {
		t.Fatalf("expected cart store to update to %s, got %s", currentStoreID, got)
	}
	if currentStoreID == originalStoreIDs[0] {
		t.Fatal("test setup expected a different current store ID")
	}
	if repo.saveCalls != 1 {
		t.Fatalf("expected one save after store change, got %d", repo.saveCalls)
	}
}
