// Package postgres implements Product persistence using PostgreSQL and SQLC.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/valueobjects"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/persistence/postgres/generated"
)

// ProductRepository persists Product aggregates, ordered images, and an optional video reference.
type ProductRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewProductRepository creates the PostgreSQL repository.
func NewProductRepository(pool *pgxpool.Pool, queries *generated.Queries) *ProductRepository {
	return &ProductRepository{pool: pool, queries: queries}
}

var _ ports.ProductRepository = (*ProductRepository)(nil)

// Create atomically inserts the product, its images, and its optional video reference.
func (r *ProductRepository) Create(ctx context.Context, product *entities.Product) error {
	if product == nil {
		return errors.New("create product: product cannot be nil")
	}

	id, err := parseUUID(product.ID())
	if err != nil {
		return fmt.Errorf("create product: invalid product ID: %w", err)
	}
	storeID, err := parseUUID(product.StoreID())
	if err != nil {
		return fmt.Errorf("create product: invalid store ID: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create product transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.queries.WithTx(tx)
	err = q.CreateProduct(ctx, generated.CreateProductParams{
		ID:                    toUUID(id),
		StoreID:               toUUID(storeID),
		Name:                  product.Name(),
		Description:           product.Description(),
		PriceAmount:           product.Price().Amount(),
		Currency:              product.Price().Currency(),
		Quantity:              product.Quantity(),
		VideoStorageReference: toVideoReference(product.Video()),
		CreatedAt:             toTimestamp(product.CreatedAt()),
		UpdatedAt:             toTimestamp(product.UpdatedAt()),
	})
	if err != nil {
		return translateWriteError("create product", err)
	}

	if err := insertImages(ctx, q, id, product.Images()); err != nil {
		return translateWriteError("create product images", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create product transaction: %w", err)
	}
	return nil
}

// GetByID returns nil, nil when the product does not exist.
func (r *ProductRepository) GetByID(ctx context.Context, productID string) (*entities.Product, error) {
	id, err := parseUUID(productID)
	if err != nil {
		return nil, fmt.Errorf("get product: invalid product ID: %w", err)
	}

	row, err := r.queries.GetProductByID(ctx, toUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get product by ID: %w", err)
	}

	images, err := r.queries.GetProductImages(ctx, toUUID(id))
	if err != nil {
		return nil, fmt.Errorf("get product images: %w", err)
	}
	return fromRow(row, images)
}

// GetByIDs returns matching products in one batch. Missing IDs are omitted.
func (r *ProductRepository) GetByIDs(ctx context.Context, productIDs []string) ([]*entities.Product, error) {
	if len(productIDs) == 0 {
		return []*entities.Product{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(productIDs))
	seen := make(map[string]struct{}, len(productIDs))
	for _, rawID := range productIDs {
		id, err := parseUUID(rawID)
		if err != nil {
			return nil, fmt.Errorf("get products by IDs: invalid product ID: %w", err)
		}
		key := id.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		ids = append(ids, toUUID(id))
	}

	rows, err := r.queries.GetProductsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get products by IDs: %w", err)
	}
	if len(rows) == 0 {
		return []*entities.Product{}, nil
	}

	imageRows, err := r.queries.GetProductImagesByProductIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get product images by IDs: %w", err)
	}
	imagesByID := make(map[string][]generated.ProductImage, len(rows))
	for _, image := range imageRows {
		if !image.ProductID.Valid {
			return nil, errors.New("get products by IDs: persisted image has NULL product ID")
		}
		key := uuid.UUID(image.ProductID.Bytes).String()
		imagesByID[key] = append(imagesByID[key], image)
	}

	products := make([]*entities.Product, 0, len(rows))
	for _, row := range rows {
		if !row.ID.Valid {
			return nil, errors.New("get products by IDs: persisted product has NULL ID")
		}
		product, err := fromRow(row, imagesByID[uuid.UUID(row.ID.Bytes).String()])
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

// Update atomically updates product fields, images, and its optional video reference.
func (r *ProductRepository) Update(ctx context.Context, product *entities.Product) error {
	if product == nil {
		return errors.New("update product: product cannot be nil")
	}

	id, err := parseUUID(product.ID())
	if err != nil {
		return fmt.Errorf("update product: invalid product ID: %w", err)
	}
	storeID, err := parseUUID(product.StoreID())
	if err != nil {
		return fmt.Errorf("update product: invalid store ID: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update product transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.queries.WithTx(tx)
	if _, err := q.GetProductByID(ctx, toUUID(id)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("update product: product %s does not exist", id)
		}
		return fmt.Errorf("check product before update: %w", err)
	}

	err = q.UpdateProduct(ctx, generated.UpdateProductParams{
		StoreID:               toUUID(storeID),
		Name:                  product.Name(),
		Description:           product.Description(),
		PriceAmount:           product.Price().Amount(),
		Currency:              product.Price().Currency(),
		Quantity:              product.Quantity(),
		VideoStorageReference: toVideoReference(product.Video()),
		UpdatedAt:             toTimestamp(product.UpdatedAt()),
		ID:                    toUUID(id),
	})
	if err != nil {
		return translateWriteError("update product", err)
	}

	if err := q.DeleteProductImages(ctx, toUUID(id)); err != nil {
		return fmt.Errorf("replace product images: delete existing images: %w", err)
	}
	if err := insertImages(ctx, q, id, product.Images()); err != nil {
		return translateWriteError("replace product images", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update product transaction: %w", err)
	}
	return nil
}

// Delete removes a product. The database cascades deletion to product_images.
func (r *ProductRepository) Delete(ctx context.Context, productID string) error {
	id, err := parseUUID(productID)
	if err != nil {
		return fmt.Errorf("delete product: invalid product ID: %w", err)
	}
	if err := r.queries.DeleteProduct(ctx, toUUID(id)); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	return nil
}

// ListByStore returns a page and the total persisted product count for a store.
func (r *ProductRepository) ListByStore(
	ctx context.Context,
	storeID string,
	limit, offset int,
) ([]*entities.Product, int64, error) {
	id, err := parseUUID(storeID)
	if err != nil {
		return nil, 0, fmt.Errorf("list products by store: invalid store ID: %w", err)
	}
	pageLimit, pageOffset, err := paginationArgs(limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.queries.CountProductsByStore(ctx, toUUID(id))
	if err != nil {
		return nil, 0, fmt.Errorf("count products by store: %w", err)
	}
	rows, err := r.queries.ListProductsByStore(ctx, generated.ListProductsByStoreParams{
		StoreID: toUUID(id), PageLimit: pageLimit, PageOffset: pageOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list products by store: %w", err)
	}
	products, err := r.loadProducts(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// CountByStore returns the current persisted product count for a store.
func (r *ProductRepository) CountByStore(ctx context.Context, storeID string) (int64, error) {
	id, err := parseUUID(storeID)
	if err != nil {
		return 0, fmt.Errorf("count products by store: invalid store ID: %w", err)
	}
	count, err := r.queries.CountProductsByStore(ctx, toUUID(id))
	if err != nil {
		return 0, fmt.Errorf("count products by store: %w", err)
	}
	return count, nil
}

// ListProducts returns a marketplace page and the total matching candidate count.
// Price sorting across different currencies is not economically comparable;
// callers should supply a price bound to constrain results to a currency.
func (r *ProductRepository) ListProducts(
	ctx context.Context,
	filters ports.ProductListFilters,
	limit, offset int,
) ([]*entities.Product, int64, error) {
	pageLimit, pageOffset, err := paginationArgs(limit, offset)
	if err != nil {
		return nil, 0, err
	}

	minCurrency, minAmount, hasMin := "", int64(0), false
	maxCurrency, maxAmount, hasMax := "", int64(0), false
	if filters.MinPrice != nil {
		hasMin = true
		minCurrency, minAmount = filters.MinPrice.Currency(), filters.MinPrice.Amount()
	}
	if filters.MaxPrice != nil {
		hasMax = true
		maxCurrency, maxAmount = filters.MaxPrice.Currency(), filters.MaxPrice.Amount()
	}

	countParams := generated.CountMarketplaceProductsParams{
		Query: filters.Query, HasMin: hasMin, MinCurrency: minCurrency,
		MinAmount: minAmount, HasMax: hasMax, MaxCurrency: maxCurrency,
		MaxAmount: maxAmount, InStockOnly: filters.InStockOnly,
	}
	total, err := r.queries.CountMarketplaceProducts(ctx, countParams)
	if err != nil {
		return nil, 0, fmt.Errorf("count marketplace products: %w", err)
	}

	rows, err := r.queries.ListMarketplaceProducts(ctx, generated.ListMarketplaceProductsParams{
		Query: filters.Query, HasMin: hasMin, MinCurrency: minCurrency,
		MinAmount: minAmount, HasMax: hasMax, MaxCurrency: maxCurrency,
		MaxAmount: maxAmount, InStockOnly: filters.InStockOnly,
		SortBy: filters.SortBy, PageLimit: pageLimit, PageOffset: pageOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list marketplace products: %w", err)
	}
	products, err := r.loadProducts(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// SearchByStore applies vendor catalogue search and exact price-bound semantics.
func (r *ProductRepository) SearchByStore(
	ctx context.Context,
	storeID string,
	query string,
	filters ports.StoreProductSearchFilters,
) ([]*entities.Product, error) {
	id, err := parseUUID(storeID)
	if err != nil {
		return nil, fmt.Errorf("search store products: invalid store ID: %w", err)
	}

	params := generated.SearchProductsByStoreParams{
		StoreID: toUUID(id), Query: query, Currency: filters.Currency,
		HasMin: filters.MinPriceAmount != nil, HasMax: filters.MaxPriceAmount != nil,
		MinInclusive: filters.MinPriceInclusive, MaxInclusive: filters.MaxPriceInclusive,
		AvailableOnly: filters.AvailableOnly,
	}
	if filters.MinPriceAmount != nil {
		params.MinAmount = *filters.MinPriceAmount
	}
	if filters.MaxPriceAmount != nil {
		params.MaxAmount = *filters.MaxPriceAmount
	}

	rows, err := r.queries.SearchProductsByStore(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search store products: %w", err)
	}
	return r.loadProducts(ctx, rows)
}

// loadProducts fetches all image rows for the supplied product page in one query.
func (r *ProductRepository) loadProducts(
	ctx context.Context,
	rows []generated.Product,
) ([]*entities.Product, error) {
	if len(rows) == 0 {
		return []*entities.Product{}, nil
	}

	ids := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		if !row.ID.Valid {
			return nil, errors.New("load products: persisted product has NULL ID")
		}
		ids = append(ids, row.ID)
	}
	imageRows, err := r.queries.GetProductImagesByProductIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load product images: %w", err)
	}

	imagesByID := make(map[string][]generated.ProductImage, len(rows))
	for _, image := range imageRows {
		if !image.ProductID.Valid {
			return nil, errors.New("load product images: persisted image has NULL product ID")
		}
		key := uuid.UUID(image.ProductID.Bytes).String()
		imagesByID[key] = append(imagesByID[key], image)
	}

	products := make([]*entities.Product, 0, len(rows))
	for _, row := range rows {
		product, err := fromRow(row, imagesByID[uuid.UUID(row.ID.Bytes).String()])
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

// fromRow reconstructs a validated domain aggregate from persisted SQLC rows.
func fromRow(row generated.Product, imageRows []generated.ProductImage) (*entities.Product, error) {
	if !row.ID.Valid || !row.StoreID.Valid {
		return nil, errors.New("reconstitute product: persisted product has a NULL UUID")
	}
	if !row.CreatedAt.Valid || !row.UpdatedAt.Valid {
		return nil, errors.New("reconstitute product: persisted product has a NULL timestamp")
	}

	price, err := valueobjects.NewMoney(row.PriceAmount, row.Currency)
	if err != nil {
		return nil, fmt.Errorf("reconstitute product price: %w", err)
	}

	var video *entities.ProductVideo
	if row.VideoStorageReference.Valid {
		video = &entities.ProductVideo{StorageReference: row.VideoStorageReference.String}
	}

	images := make([]entities.ProductImage, 0, len(imageRows))
	for _, image := range imageRows {
		images = append(images, entities.ProductImage{
			StorageReference: image.StorageReference,
		})
	}

	product, err := entities.ReconstituteProduct(
		uuid.UUID(row.ID.Bytes).String(),
		uuid.UUID(row.StoreID.Bytes).String(),
		row.Name, row.Description, price, row.Quantity,
		images, video, row.CreatedAt.Time, row.UpdatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf("reconstitute product aggregate: %w", err)
	}
	return product, nil
}

func insertImages(
	ctx context.Context,
	q *generated.Queries,
	productID uuid.UUID,
	images []entities.ProductImage,
) error {
	for position, image := range images {
		if err := q.CreateProductImage(ctx, generated.CreateProductImageParams{
			ProductID: toUUID(productID), Position: int16(position),
			StorageReference: image.StorageReference,
		}); err != nil {
			return fmt.Errorf("insert image at position %d: %w", position, err)
		}
	}
	return nil
}

func toVideoReference(video *entities.ProductVideo) pgtype.Text {
	if video == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: video.StorageReference, Valid: true}
}

func parseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

func toUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

func toTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func paginationArgs(limit, offset int) (int32, int32, error) {
	if limit <= 0 || offset < 0 || int64(limit) > math.MaxInt32 || int64(offset) > math.MaxInt32 {
		return 0, 0, errors.New("invalid product pagination values")
	}
	return int32(limit), int32(offset), nil
}

func translateWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("%s: PostgreSQL %s (%s): %w", operation, pgErr.Code, pgErr.ConstraintName, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
