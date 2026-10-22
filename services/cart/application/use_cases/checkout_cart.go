package use_cases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
	domainerrors "github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/errors"
)

// CheckoutItem combines authoritative Product data with the reconciled
// quantity requested by the customer. Product.Quantity is current stock;
// CartQuantity is the quantity that will be sent to the Order service.
type CheckoutItem struct {
	Product      ports.ProductSnapshot
	CartQuantity int64
}

// CheckoutStore groups checkout items by their current authoritative store.
type CheckoutStore struct {
	StoreID uuid.UUID
	Items   []CheckoutItem
}

// CheckoutCartOutput contains refreshed products grouped by store and the IDs
// of products removed because they no longer exist in the Product service.
type CheckoutCartOutput struct {
	Stores            []CheckoutStore
	RemovedProductIDs []uuid.UUID
}

// CheckoutCart refreshes the customer's cart against the Product service,
// persists stock/store corrections, removes missing products, and returns
// current product details grouped by store.
type CheckoutCart struct {
	carts    ports.CartRepository
	products ports.ProductClient
	logger   ports.Logger
	metrics  ports.Metrics
}

func NewCheckoutCart(
	carts ports.CartRepository,
	products ports.ProductClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *CheckoutCart {
	return &CheckoutCart{
		carts:    carts,
		products: products,
		logger:   logger,
		metrics:  metrics,
	}
}

func (uc *CheckoutCart) Execute(
	ctx context.Context,
	customerID uuid.UUID,
) (*CheckoutCartOutput, error) {
	if customerID == uuid.Nil {
		return nil, domainerrors.ErrInvalidCustomerID
	}

	cart, err := uc.carts.GetByCustomerID(ctx, customerID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrCartNotFound) {
			return nil, err
		}
		uc.recordFailure("get_cart", "persistence")
		return nil, fmt.Errorf("get customer cart for checkout: %w", err)
	}
	if cart == nil {
		uc.recordFailure("get_cart", "persistence")
		return nil, errors.New("get customer cart for checkout: repository returned a nil cart")
	}

	output := &CheckoutCartOutput{
		Stores:            make([]CheckoutStore, 0),
		RemovedProductIDs: make([]uuid.UUID, 0),
	}

	items := cart.Items()
	if len(items) == 0 {
		uc.recordSuccess()
		return output, nil
	}

	productIDs := make([]uuid.UUID, 0, len(items))
	requestedIDs := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		productIDs = append(productIDs, item.ProductID())
		requestedIDs[item.ProductID()] = struct{}{}
	}

	snapshots, err := uc.products.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		uc.recordFailure("get_products", "dependency")
		return nil, fmt.Errorf("get current products for checkout: %w", err)
	}

	// Validate the complete response before changing the cart. A malformed or
	// inconsistent response must not be mistaken for deleted products.
	snapshotByID := make(map[uuid.UUID]ports.ProductSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.ID == uuid.Nil || snapshot.StoreID == uuid.Nil || snapshot.Quantity < 0 {
			uc.recordFailure("validate_products", "dependency")
			return nil, errors.New("product service returned an invalid checkout snapshot")
		}
		if _, requested := requestedIDs[snapshot.ID]; !requested {
			uc.recordFailure("validate_products", "dependency")
			return nil, errors.New("product service returned an unrequested product")
		}
		if _, duplicate := snapshotByID[snapshot.ID]; duplicate {
			uc.recordFailure("validate_products", "dependency")
			return nil, errors.New("product service returned a duplicate product")
		}
		snapshotByID[snapshot.ID] = snapshot
	}

	changed := false
	storeIndexes := make(map[uuid.UUID]int)

	for _, item := range items {
		snapshot, exists := snapshotByID[item.ProductID()]
		if !exists {
			if err := cart.RemoveItem(item.ID()); err != nil {
				uc.recordFailure("reconcile_cart", "domain")
				return nil, fmt.Errorf("remove missing product from cart: %w", err)
			}
			output.RemovedProductIDs = append(output.RemovedProductIDs, item.ProductID())
			changed = true
			continue
		}

		cartQuantity := item.Quantity()
		if cartQuantity > snapshot.Quantity {
			cartQuantity = snapshot.Quantity
		}

		if item.StoreID() != snapshot.StoreID || item.Quantity() != cartQuantity {
			if err := cart.ReconcileItem(item.ID(), snapshot.StoreID, cartQuantity); err != nil {
				uc.recordFailure("reconcile_cart", "domain")
				return nil, fmt.Errorf("reconcile cart item: %w", err)
			}
			changed = true
		}

		storeIndex, found := storeIndexes[snapshot.StoreID]
		if !found {
			storeIndex = len(output.Stores)
			storeIndexes[snapshot.StoreID] = storeIndex
			output.Stores = append(output.Stores, CheckoutStore{
				StoreID: snapshot.StoreID,
				Items:   make([]CheckoutItem, 0),
			})
		}

		output.Stores[storeIndex].Items = append(
			output.Stores[storeIndex].Items,
			CheckoutItem{
				Product:      snapshot,
				CartQuantity: cartQuantity,
			},
		)
	}

	if changed {
		if err := uc.carts.Save(ctx, cart); err != nil {
			uc.recordFailure("save_cart", "persistence")
			return nil, fmt.Errorf("save reconciled customer cart: %w", err)
		}
	}

	uc.recordSuccess()
	return output, nil
}

func (uc *CheckoutCart) recordSuccess() {
	uc.logger.LogEvent(ports.LogEvent{
		Event:     "cart.checkout.reconciled",
		Operation: "checkout_cart",
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_checkouts_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "checkout",
			"result":    "success",
		},
	})
}

func (uc *CheckoutCart) recordFailure(stage, category string) {
	uc.logger.LogEvent(ports.LogEvent{
		Event:           "cart.checkout.failed",
		Operation:       "checkout_cart",
		FailureCategory: category,
	})
	uc.metrics.Record(ports.Metric{
		Name:  "cart_checkouts_total",
		Value: 1,
		Labels: map[string]string{
			"operation": "checkout",
			"stage":     stage,
			"result":    "failure",
		},
	})
}

// Keep the Cart entity dependency explicit in the application use-case file.
var _ *entities.Cart
