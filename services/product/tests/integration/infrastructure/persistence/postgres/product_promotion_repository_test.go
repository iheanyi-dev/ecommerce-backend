package postgres_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iheanyi-dev/ecommerce-backend/services/product/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/persistence/postgres"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/persistence/postgres/generated"
)

func newPromotionTestRepository(t *testing.T) (*postgres.ProductPromotionRepository, *pgxpool.Pool) {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("PRODUCT_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("PRODUCT_TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Product test database: %v", err)
	}
	t.Cleanup(adminPool.Close)

	schema := "product_promotion_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")

	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}

	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(),
			fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
	})

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse Product test database URL: %v", err)
	}

	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create isolated test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping isolated test pool: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE product_promotion_index (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			store_id UUID NOT NULL,
			product_id UUID NULL,
			promotion_id UUID NOT NULL,
			status VARCHAR(30) NOT NULL CHECK (LENGTH(TRIM(status)) > 0),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE UNIQUE INDEX idx_product_promotion_store_unique
			ON product_promotion_index (promotion_id)
			WHERE product_id IS NULL;

		CREATE UNIQUE INDEX idx_product_promotion_product_unique
			ON product_promotion_index (promotion_id, product_id)
			WHERE product_id IS NOT NULL;

		CREATE INDEX idx_product_promotion_store_id
			ON product_promotion_index (store_id);

		CREATE INDEX idx_product_promotion_product_lookup
			ON product_promotion_index (store_id, product_id);
	`)
	if err != nil {
		t.Fatalf("create promotion projection table: %v", err)
	}

	repository := postgres.NewProductPromotionRepository(
		pool,
		generated.New(pool),
	)

	return repository, pool
}

func newPromotion(
	t *testing.T,
	storeID string,
	productID *string,
	promotionID string,
	status string,
) *entities.ProductPromotion {
	t.Helper()

	promotion, err := entities.NewProductPromotion(
		storeID,
		productID,
		promotionID,
		status,
	)
	if err != nil {
		t.Fatalf("create promotion entity: %v", err)
	}
	return promotion
}

func TestProductPromotionRepositoryStoreWidePromotion(t *testing.T) {
	repository, _ := newPromotionTestRepository(t)
	ctx := context.Background()

	storeID := uuid.NewString()
	promotionID := uuid.NewString()
	productID := uuid.NewString()

	// Scheduled promotions must also be discoverable.
	promotion := newPromotion(t, storeID, nil, promotionID, "scheduled")

	if err := repository.Add(ctx, promotion); err != nil {
		t.Fatalf("add store-wide promotion: %v", err)
	}

	exists, err := repository.Exists(ctx, storeID, nil)
	if err != nil {
		t.Fatalf("check store-wide promotion: %v", err)
	}
	if !exists {
		t.Fatal("expected store-wide promotion to exist")
	}

	exists, err = repository.Exists(ctx, storeID, &productID)
	if err != nil {
		t.Fatalf("check product promotion: %v", err)
	}
	if !exists {
		t.Fatal("expected store-wide promotion to apply to product")
	}
}

func TestProductPromotionRepositoryProductSpecificPromotion(t *testing.T) {
	repository, _ := newPromotionTestRepository(t)
	ctx := context.Background()

	storeID := uuid.NewString()
	promotionID := uuid.NewString()
	productOne := uuid.NewString()
	productTwo := uuid.NewString()

	if err := repository.Add(ctx, newPromotion(
		t, storeID, &productOne, promotionID, "scheduled",
	)); err != nil {
		t.Fatalf("add first product promotion: %v", err)
	}

	if err := repository.Add(ctx, newPromotion(
		t, storeID, &productTwo, promotionID, "scheduled",
	)); err != nil {
		t.Fatalf("add second product promotion: %v", err)
	}

	// A store-level lookup must not report product-only promotions.
	exists, err := repository.Exists(ctx, storeID, nil)
	if err != nil {
		t.Fatalf("check store-wide promotions: %v", err)
	}
	if exists {
		t.Fatal("product-specific promotions must not count as store-wide")
	}

	exists, err = repository.Exists(ctx, storeID, &productOne)
	if err != nil {
		t.Fatalf("check first product: %v", err)
	}
	if !exists {
		t.Fatal("expected first product promotion to exist")
	}

	exists, err = repository.Exists(ctx, storeID, &productTwo)
	if err != nil {
		t.Fatalf("check second product: %v", err)
	}
	if !exists {
		t.Fatal("expected second product promotion to exist")
	}

	unrelatedProduct := uuid.NewString()
	exists, err = repository.Exists(ctx, storeID, &unrelatedProduct)
	if err != nil {
		t.Fatalf("check unrelated product: %v", err)
	}
	if exists {
		t.Fatal("unrelated product must not have a promotion")
	}
}

func TestProductPromotionRepositoryUpdateAndRemove(t *testing.T) {
	repository, pool := newPromotionTestRepository(t)
	ctx := context.Background()

	storeID := uuid.NewString()
	promotionID := uuid.NewString()
	productOne := uuid.NewString()
	productTwo := uuid.NewString()

	for _, productID := range []*string{&productOne, &productTwo} {
		if err := repository.Add(ctx, newPromotion(
			t, storeID, productID, promotionID, "scheduled",
		)); err != nil {
			t.Fatalf("add product promotion: %v", err)
		}
	}

	if err := repository.Update(ctx, newPromotion(
		t, storeID, nil, promotionID, "active",
	)); err != nil {
		t.Fatalf("update promotion: %v", err)
	}

	var total, active int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'active')
		FROM product_promotion_index
		WHERE store_id = $1 AND promotion_id = $2
	`, storeID, promotionID).Scan(&total, &active)
	if err != nil {
		t.Fatalf("read updated promotion records: %v", err)
	}

	if total != 2 || active != 2 {
		t.Fatalf("expected both records to be active; total=%d active=%d", total, active)
	}

	if err := repository.Remove(ctx, storeID, promotionID); err != nil {
		t.Fatalf("remove promotion: %v", err)
	}

	for _, productID := range []*string{&productOne, &productTwo} {
		exists, err := repository.Exists(ctx, storeID, productID)
		if err != nil {
			t.Fatalf("check removed promotion: %v", err)
		}
		if exists {
			t.Fatal("expected removed promotion to be absent")
		}
	}
}

func TestProductPromotionRepositoryRemoveDoesNotAffectOtherPromotions(t *testing.T) {
	repository, _ := newPromotionTestRepository(t)
	ctx := context.Background()

	storeID := uuid.NewString()
	productID := uuid.NewString()
	promotionOne := uuid.NewString()
	promotionTwo := uuid.NewString()

	for _, promotionID := range []string{promotionOne, promotionTwo} {
		if err := repository.Add(ctx, newPromotion(
			t, storeID, &productID, promotionID, "active",
		)); err != nil {
			t.Fatalf("add promotion: %v", err)
		}
	}

	if err := repository.Remove(ctx, storeID, promotionOne); err != nil {
		t.Fatalf("remove first promotion: %v", err)
	}

	exists, err := repository.Exists(ctx, storeID, &productID)
	if err != nil {
		t.Fatalf("check remaining promotion: %v", err)
	}
	if !exists {
		t.Fatal("removing one promotion must not remove another promotion")
	}
}
