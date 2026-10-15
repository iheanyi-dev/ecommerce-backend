// Package use_cases implements Product application workflows.
package use_cases

import (
	"errors"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

var ErrInvalidPagination = errors.New("invalid pagination")

// ProductPage contains a stable page of products and the total number of
// matching products across all pages.
type ProductPage struct {
	Products []*entities.Product
	Total    int64
	Limit    int
	Offset   int
}

func normalizePagination(limit, offset int) (int, int, error) {
	if limit < 0 || offset < 0 {
		return 0, 0, ErrInvalidPagination
	}
	if limit == 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return limit, offset, nil
}

func emptyProductPage(limit, offset int) ProductPage {
	return ProductPage{
		Products: make([]*entities.Product, 0),
		Total:    0,
		Limit:    limit,
		Offset:   offset,
	}
}

func validIdentifier(value string) bool {
	return strings.TrimSpace(value) != ""
}
