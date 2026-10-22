package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/use_cases"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
	presentationerrors "github.com/iheanyi-dev/ecommerce-backend/services/product/presentation/errors"
)

type CreateProductUseCase interface {
	Execute(context.Context, use_cases.CreateProductInput) (*entities.Product, error)
}

type GetProductUseCase interface {
	Execute(context.Context, string) (*entities.Product, error)
}

type UpdateProductUseCase interface {
	Execute(context.Context, use_cases.UpdateProductInput) (*entities.Product, error)
}

type ChangeProductImageUseCase interface {
	Execute(context.Context, use_cases.ChangeProductImageInput) (*entities.Product, error)
}

type DeleteProductUseCase interface {
	Execute(context.Context, use_cases.DeleteProductInput) error
}

type ListProductsUseCase interface {
	Execute(context.Context, use_cases.ListProductsInput) (use_cases.ProductPage, error)
}

type SearchProductsUseCase interface {
	Execute(context.Context, use_cases.SearchProductsInput) (use_cases.ProductPage, error)
}

type ListProductsByStoreUseCase interface {
	Execute(context.Context, string, int, int) (use_cases.ProductPage, error)
}

type ListMyProductsUseCase interface {
	Execute(context.Context, string, int, int) (use_cases.ProductPage, error)
}

type ProductUseCases struct {
	ChangeImage ChangeProductImageUseCase
	Create      CreateProductUseCase
	Get         GetProductUseCase
	Update      UpdateProductUseCase
	Delete      DeleteProductUseCase
	List        ListProductsUseCase
	Search      SearchProductsUseCase
	ListStore   ListProductsByStoreUseCase
	ListMine    ListMyProductsUseCase
}

type ProductHandler struct {
	uc ProductUseCases
}

func NewProductHandler(uc ProductUseCases) *ProductHandler {
	return &ProductHandler{uc: uc}
}

type productResponse struct {
	ID          string                 `json:"id"`
	StoreID     string                 `json:"store_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Price       int64                  `json:"price"`
	Currency    string                 `json:"currency"`
	Quantity    int64                  `json:"quantity"`
	Images      []productImageResponse `json:"images"`
	CreatedAt   interface{}            `json:"created_at"`
	UpdatedAt   interface{}            `json:"updated_at"`
}

type productImageResponse struct {
	StorageReference string `json:"storage_reference"`
}

func toProductResponse(product *entities.Product) productResponse {
	images := product.Images()
	imageResponses := make([]productImageResponse, 0, len(images))
	for _, image := range images {
		imageResponses = append(imageResponses, productImageResponse{
			StorageReference: image.StorageReference,
		})
	}

	return productResponse{
		ID:          product.ID(),
		StoreID:     product.StoreID(),
		Name:        product.Name(),
		Description: product.Description(),
		Price:       product.Price().Amount(),
		Currency:    product.Price().Currency(),
		Quantity:    product.Quantity(),
		Images:      imageResponses,
		CreatedAt:   product.CreatedAt(),
		UpdatedAt:   product.UpdatedAt(),
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain one JSON value")
		}
		return err
	}
	return nil
}

func authenticatedOwnerID(r *http.Request) (string, bool) {
	identity, ok := ports.AuthenticatedIdentityFromContext(r.Context())
	if !ok || identity.UserID == [16]byte{} {
		return "", false
	}
	return identity.UserID.String(), true
}

type createProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Currency    string `json:"currency"`
	Quantity    int64  `json:"quantity"`
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := authenticatedOwnerID(r)
	if !ok {
		presentationerrors.Write(w, use_cases.ErrInvalidProductOwner)
		return
	}

	var request createProductRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
		return
	}

	price, err := valueobjects.NewMoney(request.Price, request.Currency)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid price or currency"})
		return
	}

	product, err := h.uc.Create.Execute(r.Context(), use_cases.CreateProductInput{
		OwnerID:     ownerID,
		Name:        request.Name,
		Description: request.Description,
		Price:       price,
		Quantity:    request.Quantity,
	})
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProductResponse(product))
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	productID := strings.TrimSpace(r.PathValue("id"))
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product ID is required"})
		return
	}

	product, err := h.uc.Get.Execute(r.Context(), productID)
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(product))
}

type updateProductRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *int64  `json:"price"`
	Currency    *string `json:"currency"`
	Quantity    *int64  `json:"quantity"`
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := authenticatedOwnerID(r)
	if !ok {
		presentationerrors.Write(w, use_cases.ErrInvalidProductOwner)
		return
	}

	var request updateProductRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
		return
	}

	var price *valueobjects.Money
	if request.Price != nil || request.Currency != nil {
		if request.Price == nil || request.Currency == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price and currency must be provided together"})
			return
		}
		parsed, err := valueobjects.NewMoney(*request.Price, *request.Currency)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid price or currency"})
			return
		}
		price = &parsed
	}

	product, err := h.uc.Update.Execute(r.Context(), use_cases.UpdateProductInput{
		OwnerID:     ownerID,
		ProductID:   strings.TrimSpace(r.PathValue("id")),
		Name:        request.Name,
		Description: request.Description,
		Price:       price,
		Quantity:    request.Quantity,
	})
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := authenticatedOwnerID(r)
	if !ok {
		presentationerrors.Write(w, use_cases.ErrInvalidProductOwner)
		return
	}

	if err := h.uc.Delete.Execute(r.Context(), use_cases.DeleteProductInput{
		OwnerID:   ownerID,
		ProductID: strings.TrimSpace(r.PathValue("id")),
	}); err != nil {
		presentationerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parsePagination(r *http.Request) (int, int, error) {
	limit, offset := 0, 0
	var err error

	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, use_cases.ErrInvalidPagination
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, use_cases.ErrInvalidPagination
		}
	}
	return limit, offset, nil
}

type productPageResponse struct {
	Products []productResponse `json:"products"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

func toProductPageResponse(page use_cases.ProductPage) productPageResponse {
	products := make([]productResponse, 0, len(page.Products))
	for _, product := range page.Products {
		if product != nil {
			products = append(products, toProductResponse(product))
		}
	}
	return productPageResponse{
		Products: products,
		Total:    page.Total,
		Limit:    page.Limit,
		Offset:   page.Offset,
	}
}

func (h *ProductHandler) ListMarketplace(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	minPrice, maxPrice, err := parsePriceFilters(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	page, err := h.uc.List.Execute(r.Context(), use_cases.ListProductsInput{
		Query:       strings.TrimSpace(r.URL.Query().Get("q")),
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		InStockOnly: r.URL.Query().Get("in_stock") == "true",
		SortBy:      strings.TrimSpace(r.URL.Query().Get("sort")),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductPageResponse(page))
}

func (h *ProductHandler) Search(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	minPrice, maxPrice, err := parsePriceFilters(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	page, err := h.uc.Search.Execute(r.Context(), use_cases.SearchProductsInput{
		Query:       strings.TrimSpace(r.URL.Query().Get("q")),
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		InStockOnly: r.URL.Query().Get("in_stock") == "true",
		SortBy:      strings.TrimSpace(r.URL.Query().Get("sort")),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductPageResponse(page))
}

func (h *ProductHandler) ListStore(w http.ResponseWriter, r *http.Request) {
	storeID := strings.TrimSpace(r.PathValue("storeID"))
	if storeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "store ID is required"})
		return
	}
	limit, offset, err := parsePagination(r)
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}

	page, err := h.uc.ListStore.Execute(r.Context(), storeID, limit, offset)
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductPageResponse(page))
}

func (h *ProductHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := authenticatedOwnerID(r)
	if !ok {
		presentationerrors.Write(w, use_cases.ErrInvalidProductOwner)
		return
	}
	limit, offset, err := parsePagination(r)
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}

	page, err := h.uc.ListMine.Execute(r.Context(), ownerID, limit, offset)
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductPageResponse(page))
}

func parsePriceFilters(r *http.Request) (*valueobjects.Money, *valueobjects.Money, error) {
	query := r.URL.Query()
	minRaw, maxRaw := query.Get("min_price"), query.Get("max_price")
	if minRaw == "" && maxRaw == "" {
		return nil, nil, nil
	}

	currency := strings.TrimSpace(query.Get("currency"))
	if currency == "" {
		return nil, nil, errors.New("currency is required with price filters")
	}

	var minPrice, maxPrice *valueobjects.Money
	if minRaw != "" {
		amount, err := strconv.ParseInt(minRaw, 10, 64)
		if err != nil {
			return nil, nil, errors.New("min_price must be an integer amount in minor units")
		}
		money, err := valueobjects.NewMoney(amount, currency)
		if err != nil {
			return nil, nil, errors.New("invalid min_price or currency")
		}
		minPrice = &money
	}
	if maxRaw != "" {
		amount, err := strconv.ParseInt(maxRaw, 10, 64)
		if err != nil {
			return nil, nil, errors.New("max_price must be an integer amount in minor units")
		}
		money, err := valueobjects.NewMoney(amount, currency)
		if err != nil {
			return nil, nil, errors.New("invalid max_price or currency")
		}
		maxPrice = &money
	}
	return minPrice, maxPrice, nil
}

// ChangeImage adds a product image or replaces an existing image.
func (h *ProductHandler) ChangeImage(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := authenticatedOwnerID(r)
	if !ok {
		presentationerrors.Write(w, use_cases.ErrInvalidProductOwner)
		return
	}

	// MaxBytesReader limits the entire multipart request, including form overhead.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "image upload too large"})
			return
		}
		presentationerrors.Write(w, use_cases.ErrInvalidProductImage)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		presentationerrors.Write(w, use_cases.ErrInvalidProductImage)
		return
	}
	defer file.Close()

	// Detect the content type from the bytes rather than trusting the multipart header.
	header, err := io.ReadAll(io.LimitReader(file, 512))
	if err != nil || len(header) == 0 {
		presentationerrors.Write(w, use_cases.ErrInvalidProductImage)
		return
	}
	contentType := http.DetectContentType(header)
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		presentationerrors.Write(w, use_cases.ErrInvalidProductImage)
		return
	}

	product, err := h.uc.ChangeImage.Execute(r.Context(), use_cases.ChangeProductImageInput{
		OwnerID:                  ownerID,
		ProductID:                strings.TrimSpace(r.PathValue("id")),
		ExistingStorageReference: strings.TrimSpace(r.FormValue("existing_storage_reference")),
		ContentType:              contentType,
		Content:                  io.MultiReader(bytes.NewReader(header), file),
	})
	if err != nil {
		presentationerrors.Write(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toProductResponse(product))
}
