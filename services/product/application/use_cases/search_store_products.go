// Package use_cases implements Product application workflows.
package use_cases

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
)

var ErrInvalidStoreProductSearch = errors.New("invalid product search")

// SearchStoreProductsInput matches Chat's product-search filters without product_url.
type SearchStoreProductsInput struct {
	StoreID           string
	Query             string
	MinPrice          *string
	MaxPrice          *string
	Currency          string
	MinPriceInclusive bool
	MaxPriceInclusive bool
	AvailableOnly     *bool
}

// SearchStoreProductResult contains customer-safe product data for Chat.
type SearchStoreProductResult struct {
	ProductID   string
	Name        string
	Description string
	Price       string
	Currency    string
	Available   bool
}

// SearchStoreProducts searches a single store's catalogue.
type SearchStoreProducts struct {
	products ports.ProductRepository
	store    ports.StoreClient
	logger   ports.Logger
	metrics  ports.Metrics
}

func NewSearchStoreProducts(
	products ports.ProductRepository,
	store ports.StoreClient,
	logger ports.Logger,
	metrics ports.Metrics,
) *SearchStoreProducts {
	return &SearchStoreProducts{
		products: products,
		store:    store,
		logger:   logger,
		metrics:  metrics,
	}
}

func (uc *SearchStoreProducts) Execute(
	ctx context.Context,
	input SearchStoreProductsInput,
) ([]SearchStoreProductResult, error) {
	storeID := strings.TrimSpace(input.StoreID)
	query := strings.TrimSpace(input.Query)
	if storeID == "" || query == "" {
		uc.record(ctx, "product.search.validation_failed", "validation")
		return nil, fmt.Errorf("%w: store ID and query are required", ErrInvalidStoreProductSearch)
	}

	currency := strings.TrimSpace(input.Currency)
	if currency != "" {
		if _, err := valueobjects.NewMoney(0, currency); err != nil {
			uc.record(ctx, "product.search.validation_failed", "validation")
			return nil, fmt.Errorf("%w: invalid currency: %v", ErrInvalidStoreProductSearch, err)
		}
	}

	minAmount, err := parseSearchPrice(input.MinPrice, currency)
	if err != nil {
		uc.record(ctx, "product.search.validation_failed", "validation")
		return nil, fmt.Errorf("%w: minimum price: %v", ErrInvalidStoreProductSearch, err)
	}
	maxAmount, err := parseSearchPrice(input.MaxPrice, currency)
	if err != nil {
		uc.record(ctx, "product.search.validation_failed", "validation")
		return nil, fmt.Errorf("%w: maximum price: %v", ErrInvalidStoreProductSearch, err)
	}

	if (input.MinPrice != nil || input.MaxPrice != nil) && currency == "" {
		uc.record(ctx, "product.search.validation_failed", "validation")
		return nil, fmt.Errorf("%w: currency is required for price filters", ErrInvalidStoreProductSearch)
	}
	if input.MinPrice == nil && !input.MinPriceInclusive {
		uc.record(ctx, "product.search.validation_failed", "validation")
		return nil, fmt.Errorf("%w: minimum inclusivity requires a minimum price", ErrInvalidStoreProductSearch)
	}
	if input.MaxPrice == nil && !input.MaxPriceInclusive {
		uc.record(ctx, "product.search.validation_failed", "validation")
		return nil, fmt.Errorf("%w: maximum inclusivity requires a maximum price", ErrInvalidStoreProductSearch)
	}
	if minAmount != nil && maxAmount != nil && *minAmount > *maxAmount {
		uc.record(ctx, "product.search.validation_failed", "validation")
		return nil, fmt.Errorf("%w: minimum price exceeds maximum price", ErrInvalidStoreProductSearch)
	}

	active, err := uc.store.IsStoreActive(ctx, storeID)
	if err != nil {
		uc.record(ctx, "product.search.store_check_failed", "dependency")
		return nil, fmt.Errorf("check store availability: %w", err)
	}
	if !active {
		uc.record(ctx, "product.search.succeeded", "")
		return []SearchStoreProductResult{}, nil
	}

	filters := ports.StoreProductSearchFilters{
		Currency:          currency,
		MinPriceAmount:    minAmount,
		MaxPriceAmount:    maxAmount,
		MinPriceInclusive: input.MinPriceInclusive,
		MaxPriceInclusive: input.MaxPriceInclusive,
		AvailableOnly:     input.AvailableOnly != nil && *input.AvailableOnly,
	}

	products, err := uc.products.SearchByStore(ctx, storeID, query, filters)
	if err != nil {
		uc.record(ctx, "product.search.repository_failed", "persistence")
		return nil, fmt.Errorf("search products: %w", err)
	}

	results := make([]SearchStoreProductResult, 0, len(products))
	for _, product := range products {
		if product == nil {
			continue
		}
		price := product.Price()
		results = append(results, SearchStoreProductResult{
			ProductID:   product.ID(),
			Name:        product.Name(),
			Description: product.Description(),
			Price:       formatSearchPrice(price),
			Currency:    price.Currency(),
			Available:   product.Quantity() > 0,
		})
	}

	uc.record(ctx, "product.search.succeeded", "")
	return results, nil
}

// parseSearchPrice converts an exact decimal string into integer minor units.
// Currently supported currencies use either zero (JPY) or two decimal places.
func parseSearchPrice(value *string, currency string) (*int64, error) {
	if value == nil {
		return nil, nil
	}
	if currency == "" {
		return nil, errors.New("currency is required")
	}

	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil, errors.New("price cannot be empty")
	}

	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" {
		return nil, errors.New("invalid decimal price")
	}
	for _, part := range parts {
		if part == "" {
			return nil, errors.New("invalid decimal price")
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return nil, errors.New("price must be a non-negative decimal")
			}
		}
	}

	scale := 2
	if currency == "JPY" {
		scale = 0
	}

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return nil, errors.New("price is out of range")
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > scale {
		if strings.Trim(fraction[scale:], "0") != "" {
			return nil, errors.New("price has more decimal places than the currency supports")
		}
		fraction = fraction[:scale]
	}
	fraction += strings.Repeat("0", scale-len(fraction))

	multiplier := int64(100)
	if scale == 0 {
		multiplier = 1
	}
	if whole > (int64(^uint64(0)>>1)-1)/multiplier {
		return nil, errors.New("price is out of range")
	}

	amount := whole * multiplier
	if fraction != "" {
		fractionAmount, parseErr := strconv.ParseInt(fraction, 10, 64)
		if parseErr != nil || amount > int64(^uint64(0)>>1)-fractionAmount {
			return nil, errors.New("price is out of range")
		}
		amount += fractionAmount
	}

	return &amount, nil
}

func formatSearchPrice(price valueobjects.Money) string {
	if price.Currency() == "JPY" {
		return strconv.FormatInt(price.Amount(), 10)
	}

	whole := price.Amount() / 100
	fraction := price.Amount() % 100
	return fmt.Sprintf("%d.%02d", whole, fraction)
}

func (uc *SearchStoreProducts) record(ctx context.Context, event, failure string) {
	if uc.logger != nil {
		_ = uc.logger.Log(ctx, ports.LogEvent{
			Event: event, Operation: "search_products", FailureCategory: failure,
		})
	}
	if uc.metrics != nil {
		_ = uc.metrics.Increment(ctx, ports.Metric{
			Name:  "product_operations_total",
			Value: 1,
			Labels: map[string]string{
				"operation": "search_products",
				"event":     event,
			},
		})
	}
}

// Compile-time guard against accidentally losing the intended entity contract.
var _ *entities.Product
