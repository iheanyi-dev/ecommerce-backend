package grpc

import (
	"context"
	"errors"
	"fmt"

	productv1 "github.com/iheanyi-dev/ecommerce-backend/proto/product/v1"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productSearchUseCase interface {
	Execute(
		ctx context.Context,
		input use_cases.SearchStoreProductsInput,
	) ([]use_cases.SearchStoreProductResult, error)
}

type productCountUseCase interface {
	Execute(ctx context.Context, storeID string) (int64, error)
}

type productGetProductsByIDsForCartUseCase interface {
	Execute(ctx context.Context, productIDs []string) ([]use_cases.ProductResult, error)
}

// Server exposes Product's internal service-to-service API.
type Server struct {
	productv1.UnimplementedProductServiceServer

	search     productSearchUseCase
	count      productCountUseCase
	cartLookup productGetProductsByIDsForCartUseCase
}

// NewServer constructs the Product gRPC adapter.
func NewServer(
	search productSearchUseCase,
	count productCountUseCase,
) *Server {
	return &Server{search: search, count: count}
}

// NewServerWithCartLookup constructs the Product gRPC adapter with Cart lookup support.
func NewServerWithCartLookup(
	search productSearchUseCase,
	count productCountUseCase,
	cartLookup productGetProductsByIDsForCartUseCase,
) *Server {
	return &Server{
		search:     search,
		count:      count,
		cartLookup: cartLookup,
	}
}

var _ productv1.ProductServiceServer = (*Server)(nil)

func (s *Server) SearchProducts(
	ctx context.Context,
	request *productv1.SearchProductsRequest,
) (*productv1.SearchProductsResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	input := use_cases.SearchStoreProductsInput{
		StoreID:           request.GetStoreId(),
		Query:             request.GetQuery(),
		MinPriceInclusive: true,
		MaxPriceInclusive: true,
	}

	if filters := request.GetFilters(); filters != nil {
		input.MinPrice = filters.MinPrice
		input.MaxPrice = filters.MaxPrice
		input.Currency = filters.GetCurrency()
		input.AvailableOnly = filters.AvailableOnly

		if filters.MinPriceInclusive != nil {
			input.MinPriceInclusive = filters.GetMinPriceInclusive()
		}
		if filters.MaxPriceInclusive != nil {
			input.MaxPriceInclusive = filters.GetMaxPriceInclusive()
		}
	}

	results, err := s.search.Execute(ctx, input)
	if err != nil {
		if errors.Is(err, use_cases.ErrInvalidStoreProductSearch) {
			return nil, status.Error(codes.InvalidArgument, "invalid product search")
		}
		return nil, status.Error(codes.Internal, "product search failed")
	}

	response := &productv1.SearchProductsResponse{
		Products: make([]*productv1.Product, 0, len(results)),
	}
	for _, result := range results {
		response.Products = append(response.Products, &productv1.Product{
			ProductId:    result.ProductID,
			Name:         result.Name,
			Description:  result.Description,
			Price:        result.Price,
			Currency:     result.Currency,
			Available:    result.Available,
			HasPromotion: result.HasPromotion,
			// No canonical product URL is defined by the current contract.
			ProductUrl: "",
		})
	}

	return response, nil
}

func (s *Server) CountProductsByStore(
	ctx context.Context,
	request *productv1.CountProductsByStoreRequest,
) (*productv1.CountProductsByStoreResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if request.GetStoreId() == "" {
		return nil, status.Error(codes.InvalidArgument, "store_id is required")
	}

	count, err := s.count.Execute(ctx, request.GetStoreId())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("count products by store failed: %v", err))
	}

	return &productv1.CountProductsByStoreResponse{Count: count}, nil
}

// GetProductsByIDs returns existing products for internal Cart consumers.
// Missing product IDs are omitted; repository failures are returned as Internal.
func (s *Server) GetProductsByIDs(
	ctx context.Context,
	request *productv1.GetProductsByIDsRequest,
) (*productv1.GetProductsByIDsResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if len(request.GetProductIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_ids are required")
	}
	if s.cartLookup == nil {
		return nil, status.Error(codes.Internal, "cart product lookup is unavailable")
	}

	products, err := s.cartLookup.Execute(ctx, request.GetProductIds())
	if err != nil {
		if errors.Is(err, use_cases.ErrInvalidProductIDs) {
			return nil, status.Error(codes.InvalidArgument, "invalid product IDs")
		}
		return nil, status.Error(codes.Internal, "get products for cart failed")
	}

	response := &productv1.GetProductsByIDsResponse{
		Products: make([]*productv1.ProductSnapshot, 0, len(products)),
	}
	for _, product := range products {
		if product.Product == nil {
			return nil, status.Error(codes.Internal, "product lookup returned an invalid product")
		}
		response.Products = append(response.Products, &productv1.ProductSnapshot{
			ProductId:    product.ID(),
			StoreId:      product.StoreID(),
			Name:         product.Name(),
			Description:  product.Description(),
			PriceMinor:   product.Price().Amount(),
			Currency:     product.Price().Currency(),
			Quantity:     product.Quantity(),
			HasPromotion: product.HasPromotion,
		})
	}

	return response, nil
}
