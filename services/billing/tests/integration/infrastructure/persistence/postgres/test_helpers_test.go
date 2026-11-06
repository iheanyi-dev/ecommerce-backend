package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/billing/domain/value_objects"
	billingmigrations "github.com/iheanyi-dev/ecommerce-backend/services/billing/migrations"
)

func newBillingTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	_ = godotenv.Load("../../../../../.env")

	databaseURL := strings.TrimSpace(os.Getenv("BILLING_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("BILLING_TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse Billing test database URL: %v", err)
	}

	databaseName := poolConfig.ConnConfig.Database
	if databaseName == "billing" || !strings.HasSuffix(databaseName, "_test") {
		t.Fatalf("refusing to run integration tests against database %q; a dedicated *_test database is required", databaseName)
	}

	adminPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("connect to Billing test database: %v", err)
	}

	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		adminPool.Close()
		t.Fatalf("generate isolated schema name: %v", err)
	}

	schema := "billing_it_" + hex.EncodeToString(randomBytes[:])
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		adminPool.Close()
		t.Fatalf("create isolated Billing schema: %v", err)
	}

	appPoolConfig := poolConfig.Copy()
	appPoolConfig.ConnConfig.RuntimeParams["search_path"] = schema

	appPool, err := pgxpool.NewWithConfig(ctx, appPoolConfig)
	if err != nil {
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		adminPool.Close()
		t.Fatalf("create isolated Billing pool: %v", err)
	}

	migrationDB := stdlib.OpenDBFromPool(appPool)
	if err := goose.SetDialect("postgres"); err != nil {
		migrationDB.Close()
		appPool.Close()
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		adminPool.Close()
		t.Fatalf("configure Goose: %v", err)
	}

	goose.SetBaseFS(billingmigrations.FS)
	if err := goose.UpContext(ctx, migrationDB, "."); err != nil {
		migrationDB.Close()
		appPool.Close()
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		adminPool.Close()
		t.Fatalf("apply Billing migrations: %v", err)
	}
	if err := migrationDB.Close(); err != nil {
		t.Fatalf("close migration database: %v", err)
	}

	t.Cleanup(func() {
		appPool.Close()

		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		if _, err := adminPool.Exec(
			cleanupCtx,
			"DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE",
		); err != nil {
			t.Errorf("drop isolated Billing schema: %v", err)
		}
		adminPool.Close()
	})

	return appPool
}

func newBillingTestPlan(t *testing.T) entities.Plan {
	t.Helper()

	price, err := value_objects.NewMoney(1500000, "NGN")
	if err != nil {
		t.Fatalf("create plan price: %v", err)
	}

	now := time.Now().UTC()
	return entities.Plan{
		ID:          uuid.New(),
		Type:        "premium-" + uuid.NewString(),
		Name:        "Premium",
		Description: "Integration test plan",
		Price:       price,
		Duration:    30 * 24 * time.Hour,
		Capabilities: entities.PlanCapabilities{
			MaxProducts: 5,
			AIEnabled:   true,
		},
		Available: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newBillingTestSubscription(t *testing.T, plan entities.Plan) entities.Subscription {
	t.Helper()

	price, err := value_objects.NewMoney(1500000, "NGN")
	if err != nil {
		t.Fatalf("create subscription price: %v", err)
	}

	now := time.Now().UTC()
	return entities.Subscription{
		ID:        uuid.New(),
		StoreID:   uuid.New(),
		PlanID:    plan.ID,
		Price:     price,
		Duration:  30 * 24 * time.Hour,
		Status:    entities.SubscriptionStatus("pending"),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
