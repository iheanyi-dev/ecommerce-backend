package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/iheanyi-dev/ecommerce-backend/services/order/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/order/domain/entities"
	orderpostgres "github.com/iheanyi-dev/ecommerce-backend/services/order/infrastructure/persistence/postgres"
)

const integrationDatabaseName = "order"
const integrationDatabaseUser = "order"

func TestPostgresUnitOfWorkIntegration(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()

	t.Run("commits order and product snapshots atomically", func(t *testing.T) {
		order := newIntegrationOrder(t)
		product := newIntegrationProduct(order.ID, 2)

		registerOrderCleanup(t, pool, order.ID)

		uow := orderpostgres.NewUnitOfWork(pool)
		err := uow.WithinTransaction(ctx, func(
			orders ports.OrderRepository,
			products ports.OrderProductRepository,
		) error {
			if err := orders.CreateMany(ctx, []*entities.Order{order}); err != nil {
				return err
			}
			return products.CreateMany(ctx, []*entities.OrderProduct{&product})
		})
		if err != nil {
			t.Fatalf("commit order transaction: %v", err)
		}

		orderRepository := orderpostgres.NewOrderRepository(pool)
		savedOrder, err := orderRepository.FindByID(ctx, order.ID)
		if err != nil {
			t.Fatalf("find committed order: %v", err)
		}

		if savedOrder.ID != order.ID ||
			savedOrder.CustomerID != order.CustomerID ||
			savedOrder.StoreID != order.StoreID ||
			savedOrder.State != entities.OrderStateCreated ||
			savedOrder.PriceProduct != order.PriceProduct {
			t.Fatalf("saved order differs from fixture: got %+v", savedOrder)
		}

		productRepository := orderpostgres.NewOrderProductRepository(pool)
		savedProducts, err := productRepository.FindByOrderIDs(
			ctx,
			[]uuid.UUID{order.ID},
		)
		if err != nil {
			t.Fatalf("find committed product snapshots: %v", err)
		}
		if len(savedProducts) != 1 {
			t.Fatalf("expected 1 product snapshot, got %d", len(savedProducts))
		}
		if savedProducts[0].ID != product.ID ||
			savedProducts[0].OrderID != order.ID ||
			savedProducts[0].ProductID != product.ProductID ||
			savedProducts[0].Qty != product.Qty ||
			savedProducts[0].Price != product.Price {
			t.Fatalf("saved product differs from fixture: got %+v", savedProducts[0])
		}
	})

	t.Run("rolls back order when product insertion fails", func(t *testing.T) {
		order := newIntegrationOrder(t)
		invalidProduct := newIntegrationProduct(order.ID, 0)

		registerOrderCleanup(t, pool, order.ID)

		uow := orderpostgres.NewUnitOfWork(pool)
		err := uow.WithinTransaction(ctx, func(
			orders ports.OrderRepository,
			products ports.OrderProductRepository,
		) error {
			if err := orders.CreateMany(ctx, []*entities.Order{order}); err != nil {
				return err
			}
			return products.CreateMany(ctx, []*entities.OrderProduct{&invalidProduct})
		})
		if err == nil {
			t.Fatal("expected invalid product quantity to fail")
		}

		assertOrderAbsent(t, pool, order.ID)
		assertProductsAbsent(t, pool, order.ID)
	})

	t.Run("rolls back all writes when callback fails", func(t *testing.T) {
		order := newIntegrationOrder(t)
		product := newIntegrationProduct(order.ID, 1)
		registerOrderCleanup(t, pool, order.ID)

		expectedErr := errors.New("deliberate integration-test rollback")
		uow := orderpostgres.NewUnitOfWork(pool)
		err := uow.WithinTransaction(ctx, func(
			orders ports.OrderRepository,
			products ports.OrderProductRepository,
		) error {
			if err := orders.CreateMany(ctx, []*entities.Order{order}); err != nil {
				return err
			}
			if err := products.CreateMany(ctx, []*entities.OrderProduct{&product}); err != nil {
				return err
			}
			return expectedErr
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected callback error to be preserved, got %v", err)
		}

		assertOrderAbsent(t, pool, order.ID)
		assertProductsAbsent(t, pool, order.ID)
	})
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("ORDER_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set ORDER_TEST_DATABASE_URL to opt in to Order PostgreSQL integration tests")
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal("ORDER_TEST_DATABASE_URL is not a valid URL")
	}
	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		t.Fatal("ORDER_TEST_DATABASE_URL must use the PostgreSQL URL scheme")
	}
	if parsedURL.Path != "/"+integrationDatabaseName {
		t.Fatalf("integration tests only permit the %q database", integrationDatabaseName)
	}
	if parsedURL.User == nil || parsedURL.User.Username() != integrationDatabaseUser {
		t.Fatalf("integration tests only permit the %q database role", integrationDatabaseUser)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not create PostgreSQL connection pool")
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal("could not connect to the dedicated Order test database")
	}

	applyOrderMigrations(t, databaseURL)
	return pool
}

func applyOrderMigrations(t *testing.T, databaseURL string) {
	t.Helper()

	root, err := findRepositoryRoot()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal("could not open migration database connection")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatal("could not connect to Order database for migrations")
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("configure Goose PostgreSQL dialect: %v", err)
	}

	migrationDirectory := filepath.Join(root, "services", "order", "migrations")
	if err := goose.UpContext(ctx, db, migrationDirectory); err != nil {
		t.Fatalf("apply Order migrations: %v", err)
	}
}

func findRepositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not locate go.mod from current directory")
		}
		current = parent
	}
}

func newIntegrationOrder(t *testing.T) *entities.Order {
	t.Helper()

	order, err := entities.NewOrder(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		"Integration test delivery location",
		"NGN",
		125000,
	)
	if err != nil {
		t.Fatalf("create order fixture: %v", err)
	}
	return order
}

func newIntegrationProduct(orderID uuid.UUID, quantity int64) entities.OrderProduct {
	return entities.OrderProduct{
		ID:        uuid.New(),
		OrderID:   orderID,
		ProductID: uuid.New(),
		Name:      "Integration test product",
		Qty:       quantity,
		Price:     62500,
	}
}

func registerOrderCleanup(t *testing.T, pool *pgxpool.Pool, orderID uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Delete only this test's rows; never truncate or drop the schema.
		_, productErr := pool.Exec(
			ctx,
			`DELETE FROM order_products WHERE order_id = $1`,
			orderID,
		)
		_, orderErr := pool.Exec(
			ctx,
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)

		if productErr != nil {
			t.Errorf("clean up test product snapshots: %v", productErr)
		}
		if orderErr != nil {
			t.Errorf("clean up test order: %v", orderErr)
		}
	})
}

func assertOrderAbsent(t *testing.T, pool *pgxpool.Pool, orderID uuid.UUID) {
	t.Helper()

	var count int
	err := pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM orders WHERE id = $1`,
		orderID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("check order rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected order %s to be rolled back, found %d rows", orderID, count)
	}
}

func assertProductsAbsent(t *testing.T, pool *pgxpool.Pool, orderID uuid.UUID) {
	t.Helper()

	var count int
	err := pool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM order_products WHERE order_id = $1`,
		orderID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("check product snapshot rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected product snapshots for order %s to be rolled back, found %d rows", orderID, count)
	}
}
