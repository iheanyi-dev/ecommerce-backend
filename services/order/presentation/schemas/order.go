package schemas

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
)

type CreateOrderRequest struct {
	CartID           uuid.UUID `json:"cart_id"`
	DeliveryLocation string    `json:"delivery_location"`
}

func (r CreateOrderRequest) Validate() bool {
	return r.CartID != uuid.Nil && strings.TrimSpace(r.DeliveryLocation) != ""
}

type AddShippingRequest struct {
	ShippingAmount int64 `json:"shipping_amount_minor"`
}

func (r AddShippingRequest) Validate() bool {
	return r.ShippingAmount >= 0
}

type OrderProductResponse struct {
	ID         uuid.UUID `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	ProductID  uuid.UUID `json:"product_id"`
	Name       string    `json:"name"`
	Quantity   int64     `json:"quantity"`
	PriceMinor int64     `json:"price_minor"`
}

type OrderResponse struct {
	ID               uuid.UUID              `json:"id"`
	CartID           uuid.UUID              `json:"cart_id"`
	StoreID          uuid.UUID              `json:"store_id"`
	CustomerID       uuid.UUID              `json:"customer_id"`
	DeliveryLocation string                 `json:"delivery_location"`
	PriceProduct     int64                  `json:"price_product_minor"`
	PriceShipping    int64                  `json:"price_shipping_minor"`
	TotalPrice       int64                  `json:"total_price_minor"`
	Currency         string                 `json:"currency"`
	State            entities.OrderState    `json:"state"`
	Products         []OrderProductResponse `json:"products"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type OrderPageResponse struct {
	Items      []OrderResponse `json:"items"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalItems int64           `json:"total_items"`
	TotalPages int64           `json:"total_pages"`
}

func FromDetails(details *use_cases.OrderDetails) OrderResponse {
	order := details.Order
	total, _ := order.TotalPrice()

	products := make([]OrderProductResponse, 0, len(details.Products))
	for _, product := range details.Products {
		products = append(products, OrderProductResponse{
			ID: product.ID, OrderID: product.OrderID,
			ProductID: product.ProductID, Name: product.Name,
			Quantity: product.Qty, PriceMinor: product.Price,
		})
	}

	return OrderResponse{
		ID: order.ID, CartID: order.CartID, StoreID: order.StoreID,
		CustomerID: order.CustomerID, DeliveryLocation: order.DeliveryLocation,
		PriceProduct: order.PriceProduct, PriceShipping: order.PriceShipping,
		TotalPrice: total, Currency: order.Currency, State: order.State,
		Products: products, CreatedAt: order.CreatedAt, UpdatedAt: order.UpdatedAt,
	}
}
