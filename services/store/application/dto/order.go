package dto

import (
	"time"

	"github.com/google/uuid"
)

type OrderProductOutput struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	ProductID  uuid.UUID
	Name       string
	Quantity   int64
	PriceMinor int64
}

type OrderOutput struct {
	ID               uuid.UUID
	CartID           uuid.UUID
	StoreID          uuid.UUID
	CustomerID       uuid.UUID
	DeliveryLocation string
	PriceProduct     int64
	PriceShipping    int64
	Currency         string
	State            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Products         []OrderProductOutput
}

type ListStoreOrdersOutput struct {
	Orders     []OrderOutput
	Page       int
	PageSize   int
	TotalItems int64
	TotalPages int64
}
