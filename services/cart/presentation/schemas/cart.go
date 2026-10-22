package schemas

import (
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/cart/domain/entities"
)

// AddItemRequest describes the requested product and quantity.
type AddItemRequest struct {
	ProductID string `json:"product_id"`
	StoreID   string `json:"store_id"`
	Quantity  int64  `json:"quantity"`
}

// SetItemQuantityRequest contains the requested absolute quantity.
type SetItemQuantityRequest struct {
	Quantity int64 `json:"quantity"`
}

// CartResponse is the public HTTP representation of a cart.
type CartResponse struct {
	ID         string             `json:"id"`
	CustomerID string             `json:"customer_id"`
	Items      []CartItemResponse `json:"items"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// CartItemResponse is the public HTTP representation of a cart item.
type CartItemResponse struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	StoreID   string `json:"store_id"`
	Quantity  int64  `json:"quantity"`
}

// NewCartResponse maps a domain cart into its transport representation.
func NewCartResponse(cart *entities.Cart) CartResponse {
	items := cart.Items()
	response := CartResponse{
		ID:         cart.ID().String(),
		CustomerID: cart.CustomerID().String(),
		Items:      make([]CartItemResponse, 0, len(items)),
		CreatedAt:  cart.CreatedAt(),
		UpdatedAt:  cart.UpdatedAt(),
	}

	for _, item := range items {
		response.Items = append(response.Items, CartItemResponse{
			ID:        item.ID().String(),
			ProductID: item.ProductID().String(),
			StoreID:   item.StoreID().String(),
			Quantity:  item.Quantity(),
		})
	}

	return response
}

// CheckoutResponse contains reconciled checkout data.
type CheckoutResponse struct {
	Stores            []CheckoutStoreResponse `json:"stores"`
	RemovedProductIDs []string                `json:"removed_product_ids"`
}

// CheckoutStoreResponse groups products by store.
type CheckoutStoreResponse struct {
	StoreID string                 `json:"store_id"`
	Items   []CheckoutItemResponse `json:"items"`
}

// CheckoutItemResponse contains authoritative product data and cart quantity.
type CheckoutItemResponse struct {
	Product      ProductResponse `json:"product"`
	CartQuantity int64           `json:"cart_quantity"`
}

// ProductResponse is the Product service snapshot exposed to the client.
type ProductResponse struct {
	ID          string `json:"id"`
	StoreID     string `json:"store_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceMinor  int64  `json:"price_minor"`
	Currency    string `json:"currency"`
	Quantity    int64  `json:"quantity"`
}

// NewCheckoutResponse maps the checkout use-case output into an HTTP response.
func NewCheckoutResponse(output *use_cases.CheckoutCartOutput) CheckoutResponse {
	response := CheckoutResponse{
		Stores:            make([]CheckoutStoreResponse, 0, len(output.Stores)),
		RemovedProductIDs: make([]string, 0, len(output.RemovedProductIDs)),
	}

	for _, store := range output.Stores {
		storeResponse := CheckoutStoreResponse{
			StoreID: store.StoreID.String(),
			Items:   make([]CheckoutItemResponse, 0, len(store.Items)),
		}

		for _, item := range store.Items {
			product := item.Product
			storeResponse.Items = append(storeResponse.Items, CheckoutItemResponse{
				Product: ProductResponse{
					ID:          product.ID.String(),
					StoreID:     product.StoreID.String(),
					Name:        product.Name,
					Description: product.Description,
					PriceMinor:  product.PriceMinor,
					Currency:    product.Currency,
					Quantity:    product.Quantity,
				},
				CartQuantity: item.CartQuantity,
			})
		}

		response.Stores = append(response.Stores, storeResponse)
	}

	for _, id := range output.RemovedProductIDs {
		response.RemovedProductIDs = append(response.RemovedProductIDs, id.String())
	}

	return response
}

// ParseUUID parses a UUID supplied in an HTTP path or body.
func ParseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}
