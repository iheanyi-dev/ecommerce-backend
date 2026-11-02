package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/enums"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	paymentpostgres "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres"
	paymentmigrations "github.com/iheanyi-dev/ecommerce-backend/services/payment/migrations"
)

func TestUnitOfWorkIntegration(t *testing.T) {
	_ = godotenv.Load("services/payment/.env")

	databaseURL := strings.TrimSpace(os.Getenv("PAYMENT_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("PAYMENT_TEST_DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}

	databaseName := poolConfig.ConnConfig.Database
	if databaseName == "payment" || !strings.HasSuffix(databaseName, "_test") {
		t.Fatalf("refusing to run integration tests against database %q; a dedicated *_test database is required", databaseName)
	}

	adminPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer adminPool.Close()

	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		t.Fatalf("generate isolated schema name: %v", err)
	}

	schema := "payment_it_" + hex.EncodeToString(randomBytes[:])

	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}

	var testPool *pgxpool.Pool

	defer func() {
		if testPool != nil {
			testPool.Close()
		}

		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cleanupCancel()

		if _, err := adminPool.Exec(
			cleanupCtx,
			"DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE",
		); err != nil {
			t.Errorf("drop isolated test schema: %v", err)
		}
	}()

	appPoolConfig := poolConfig.Copy()
	appPoolConfig.ConnConfig.RuntimeParams["search_path"] = schema

	testPool, err = pgxpool.NewWithConfig(ctx, appPoolConfig)
	if err != nil {
		t.Fatalf("connect using isolated test schema: %v", err)
	}

	migrationDB := stdlib.OpenDBFromPool(testPool)
	defer migrationDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("configure migration dialect: %v", err)
	}
	goose.SetBaseFS(paymentmigrations.FS)

	if err := goose.UpContext(ctx, migrationDB, "."); err != nil {
		t.Fatalf("apply payment migrations to isolated schema: %v", err)
	}

	repository, err := paymentpostgres.NewPaymentRepository(testPool)
	if err != nil {
		t.Fatalf("create payment repository: %v", err)
	}

	unitOfWork, err := paymentpostgres.NewUnitOfWork(testPool)
	if err != nil {
		t.Fatalf("create unit of work: %v", err)
	}

	t.Run("commits payment", func(t *testing.T) {
		payment := newTestPayment(t, "commit")
		err := unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
			return repository.Save(txCtx, payment)
		})
		if err != nil {
			t.Fatalf("commit payment: %v", err)
		}

		saved, err := repository.GetByID(ctx, payment.ID())
		if err != nil {
			t.Fatalf("get committed payment: %v", err)
		}
		if saved == nil {
			t.Fatal("expected committed payment to exist")
		}
		if saved.ID() != payment.ID() {
			t.Fatalf("expected payment ID %q, got %q", payment.ID(), saved.ID())
		}
		if saved.Amount().Amount() != payment.Amount().Amount() {
			t.Fatalf("expected amount %d, got %d", payment.Amount().Amount(), saved.Amount().Amount())
		}
	})

	t.Run("rolls back payment when callback fails", func(t *testing.T) {
		payment := newTestPayment(t, "rollback")
		expectedErr := errors.New("deliberate transaction failure")

		err := unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := repository.Save(txCtx, payment); err != nil {
				return err
			}
			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected callback error %v, got %v", expectedErr, err)
		}

		saved, err := repository.GetByID(ctx, payment.ID())
		if err != nil {
			t.Fatalf("check rolled-back payment: %v", err)
		}
		if saved != nil {
			t.Fatal("payment exists despite transaction rollback")
		}
	})

	t.Run("nested transaction reuses outer transaction", func(t *testing.T) {
		first := newTestPayment(t, "nested-first")
		second := newTestPayment(t, "nested-second")
		expectedErr := errors.New("deliberate outer transaction failure")

		err := unitOfWork.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := repository.Save(txCtx, first); err != nil {
				return err
			}

			if err := unitOfWork.WithinTransaction(txCtx, func(nestedCtx context.Context) error {
				return repository.Save(nestedCtx, second)
			}); err != nil {
				return fmt.Errorf("nested transaction: %w", err)
			}

			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected outer callback error %v, got %v", expectedErr, err)
		}

		for _, payment := range []*entities.Payment{first, second} {
			saved, err := repository.GetByID(ctx, payment.ID())
			if err != nil {
				t.Fatalf("check payment %q: %v", payment.ID(), err)
			}
			if saved != nil {
				t.Fatalf("payment %q exists despite outer transaction rollback", payment.ID())
			}
		}
	})
}

func newTestPayment(t *testing.T, suffix string) *entities.Payment {
	t.Helper()

	payerID := "00000000-0000-4000-8000-000000000001"

	reference, err := value_objects.NewBusinessReference(
		"order",
		"integration-"+suffix,
	)
	if err != nil {
		t.Fatalf("create business reference: %v", err)
	}

	amount, err := value_objects.NewMoney(125000, "NGN")
	if err != nil {
		t.Fatalf("create payment amount: %v", err)
	}

	payment, err := entities.NewPayment(
		payerID,
		enums.PaymentPurposeOrder,
		reference,
		amount,
	)
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	return payment
}
