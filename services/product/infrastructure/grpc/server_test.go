package grpc

import (
	"context"
	"errors"
	"testing"

	productv1 "github.com/iheanyi-dev/ecommerce-backend/proto/product/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type searchUseCaseMock struct {
	input   use_cases.SearchStoreProductsInput
	results []use_cases.SearchStoreProductResult
	err     error
	calls   int
}

func (m *searchUseCaseMock) Execute(
	_ context.Context,
	input use_cases.SearchStoreProductsInput,
) ([]use_cases.SearchStoreProductResult, error) {
	m.calls++
	m.input = input
	return m.results, m.err
}

type countUseCaseMock struct {
	storeID string
	count   int64
	err     error
	calls   int
}

func (m *countUseCaseMock) Execute(
	_ context.Context,
	storeID string,
) (int64, error) {
	m.calls++
	m.storeID = storeID
	return m.count, m.err
}

func TestSearchProductsMapsRequestAndResponse(t *testing.T) {
	minPrice := "10.50"
	maxPrice := "99.99"
	minInclusive := false
	maxInclusive := true
	availableOnly := false

	search := &searchUseCaseMock{
		results: []use_cases.SearchStoreProductResult{
			{
				ProductID:   "product-123",
				Name:        "Running Shoes",
				Description: "Lightweight shoes",
				Price:       "49.99",
				Currency:    "USD",
				Available:   true,
			},
		},
	}
	server := NewServer(search, &countUseCaseMock{})

	response, err := server.SearchProducts(
		context.Background(),
		&productv1.SearchProductsRequest{
			StoreId: "store-123",
			Query:   "running shoes",
			Filters: &productv1.ProductFilters{
				MinPrice:          &minPrice,
				MaxPrice:          &maxPrice,
				Currency:          "USD",
				MinPriceInclusive: &minInclusive,
				MaxPriceInclusive: &maxInclusive,
				AvailableOnly:     &availableOnly,
			},
		},
	)
	if err != nil {
		t.Fatalf("SearchProducts() error = %v", err)
	}

	if search.calls != 1 {
		t.Fatalf("search use case calls = %d, want 1", search.calls)
	}

	input := search.input
	if input.StoreID != "store-123" || input.Query != "running shoes" {
		t.Fatalf("unexpected search input: %+v", input)
	}
	if input.MinPrice == nil || *input.MinPrice != minPrice {
		t.Fatalf("MinPrice = %v, want %q", input.MinPrice, minPrice)
	}
	if input.MaxPrice == nil || *input.MaxPrice != maxPrice {
		t.Fatalf("MaxPrice = %v, want %q", input.MaxPrice, maxPrice)
	}
	if input.Currency != "USD" {
		t.Fatalf("Currency = %q, want USD", input.Currency)
	}
	if input.MinPriceInclusive || !input.MaxPriceInclusive {
		t.Fatalf("unexpected inclusive bounds: %+v", input)
	}
	if input.AvailableOnly == nil || *input.AvailableOnly {
		t.Fatalf("AvailableOnly = %v, want pointer to false", input.AvailableOnly)
	}

	if len(response.GetProducts()) != 1 {
		t.Fatalf("response products = %d, want 1", len(response.GetProducts()))
	}

	got := response.GetProducts()[0]
	if got.GetProductId() != "product-123" ||
		got.GetName() != "Running Shoes" ||
		got.GetDescription() != "Lightweight shoes" ||
		got.GetPrice() != "49.99" ||
		got.GetCurrency() != "USD" ||
		!got.GetAvailable() {
		t.Fatalf("unexpected product response: %+v", got)
	}
	if got.GetProductUrl() != "" {
		t.Fatalf("ProductUrl = %q, want empty until canonical URL is defined", got.GetProductUrl())
	}
}

func TestSearchProductsWithoutFilters(t *testing.T) {
	search := &searchUseCaseMock{}
	server := NewServer(search, &countUseCaseMock{})

	response, err := server.SearchProducts(
		context.Background(),
		&productv1.SearchProductsRequest{
			StoreId: "store-123",
			Query:   "headphones",
		},
	)
	if err != nil {
		t.Fatalf("SearchProducts() error = %v", err)
	}
	if response == nil || len(response.GetProducts()) != 0 {
		t.Fatalf("expected empty products response, got %+v", response)
	}
	if search.input.StoreID != "store-123" || search.input.Query != "headphones" {
		t.Fatalf("unexpected search input: %+v", search.input)
	}
	if !search.input.MinPriceInclusive || !search.input.MaxPriceInclusive {
		t.Fatalf("default price boundaries should be inclusive: %+v", search.input)
	}
	if search.input.AvailableOnly != nil {
		t.Fatalf("AvailableOnly = %v, want nil when unspecified", search.input.AvailableOnly)
	}
}

func TestSearchProductsRejectsNilRequest(t *testing.T) {
	server := NewServer(&searchUseCaseMock{}, &countUseCaseMock{})

	_, err := server.SearchProducts(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestSearchProductsMapsUseCaseErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{
			name: "invalid search",
			err:  use_cases.ErrInvalidStoreProductSearch,
			want: codes.InvalidArgument,
		},
		{
			name: "unexpected failure",
			err:  errors.New("repository unavailable"),
			want: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(
				&searchUseCaseMock{err: tt.err},
				&countUseCaseMock{},
			)

			_, err := server.SearchProducts(
				context.Background(),
				&productv1.SearchProductsRequest{
					StoreId: "store-123",
					Query:   "laptop",
				},
			)
			if status.Code(err) != tt.want {
				t.Fatalf("status code = %v, want %v", status.Code(err), tt.want)
			}
		})
	}
}

func TestCountProductsByStoreReturnsCount(t *testing.T) {
	count := &countUseCaseMock{count: 7}
	server := NewServer(&searchUseCaseMock{}, count)

	response, err := server.CountProductsByStore(
		context.Background(),
		&productv1.CountProductsByStoreRequest{StoreId: "store-123"},
	)
	if err != nil {
		t.Fatalf("CountProductsByStore() error = %v", err)
	}
	if response.GetCount() != 7 {
		t.Fatalf("count = %d, want 7", response.GetCount())
	}
	if count.calls != 1 || count.storeID != "store-123" {
		t.Fatalf("unexpected count use case invocation: %+v", count)
	}
}

func TestCountProductsByStoreRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name    string
		request *productv1.CountProductsByStoreRequest
	}{
		{name: "nil request", request: nil},
		{name: "missing store ID", request: &productv1.CountProductsByStoreRequest{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := &countUseCaseMock{}
			server := NewServer(&searchUseCaseMock{}, count)

			_, err := server.CountProductsByStore(context.Background(), tt.request)
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("status code = %v, want InvalidArgument", status.Code(err))
			}
			if count.calls != 0 {
				t.Fatalf("count use case called %d times for invalid request", count.calls)
			}
		})
	}
}

func TestCountProductsByStoreMapsUseCaseError(t *testing.T) {
	server := NewServer(
		&searchUseCaseMock{},
		&countUseCaseMock{err: errors.New("database unavailable")},
	)

	_, err := server.CountProductsByStore(
		context.Background(),
		&productv1.CountProductsByStoreRequest{StoreId: "store-123"},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want Internal", status.Code(err))
	}
}

func TestGetProductsByIDsRejectsNilRequest(t *testing.T) {
	server := NewServer(nil, nil)

	_, err := server.GetProductsByIDs(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestGetProductsByIDsRejectsEmptyIDs(t *testing.T) {
	server := NewServer(nil, nil)

	_, err := server.GetProductsByIDs(
		context.Background(),
		&productv1.GetProductsByIDsRequest{},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestGetProductsByIDsRequiresCartLookup(t *testing.T) {
	server := NewServer(nil, nil)

	_, err := server.GetProductsByIDs(
		context.Background(),
		&productv1.GetProductsByIDsRequest{
			ProductIds: []string{"233b7e6c-8920-4861-a778-6f11917db316"},
		},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}
