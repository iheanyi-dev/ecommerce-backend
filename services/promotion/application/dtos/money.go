// Package dtos defines data transfer objects used by the Promotion application layer.
package dtos

// MoneyDTO represents a monetary amount in the smallest currency unit.
type MoneyDTO struct {
	Amount   int64
	Currency string
}
