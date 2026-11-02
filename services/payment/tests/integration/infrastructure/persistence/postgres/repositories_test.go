package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/domain/value_objects"
	postgres "github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres"
	paymentmigrations "github.com/iheanyi-dev/ecommerce-backend/services/payment/migrations"
)

func TestPostgresRepositoriesIntegration(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	ctx := context.Background()

	paymentRepository, err := postgres.NewPaymentRepository(pool)
	if err != nil {
		t.Fatalf("create payment repository: %v", err)
	}

	paymentAttemptRepository, err := postgres.NewPaymentAttemptRepository(pool)
	if err != nil {
		t.Fatalf("create payment attempt repository: %v", err)
	}

	encryptionKey := []byte("0123456789abcdef0123456789abcdef")
	encryptor, err := postgres.NewBeneficiaryEncryptor(encryptionKey)
	if err != nil {
		t.Fatalf("create beneficiary encryptor: %v", err)
	}

	payoutRepository, err := postgres.NewPayoutRepository(pool, encryptor)
	if err != nil {
		t.Fatalf("create payout repository: %v", err)
	}

	payoutAttemptRepository, err := postgres.NewPayoutAttemptRepository(pool)
	if err != nil {
		t.Fatalf("create payout attempt repository: %v", err)
	}

	t.Run("payment attempts", func(t *testing.T) {
		payment := newTestPayment(t, "repository-attempt")
		if err := paymentRepository.Save(ctx, payment); err != nil {
			t.Fatalf("save parent payment: %v", err)
		}

		attempt, err := entities.NewPaymentAttempt(payment.ID(), "paystack")
		if err != nil {
			t.Fatalf("create payment attempt: %v", err)
		}

		if err := paymentAttemptRepository.Save(ctx, attempt); err != nil {
			t.Fatalf("save payment attempt: %v", err)
		}

		got, err := paymentAttemptRepository.GetByID(ctx, attempt.ID())
		if err != nil {
			t.Fatalf("get payment attempt: %v", err)
		}
		if got == nil {
			t.Fatal("expected payment attempt, got nil")
		}
		if got.ID() != attempt.ID() {
			t.Fatalf("payment attempt ID = %q, want %q", got.ID(), attempt.ID())
		}
		if got.PaymentID() != payment.ID() {
			t.Fatalf("payment ID = %q, want %q", got.PaymentID(), payment.ID())
		}
		if got.Provider() != attempt.Provider() {
			t.Fatalf("provider = %q, want %q", got.Provider(), attempt.Provider())
		}
	})

	t.Run("payouts", func(t *testing.T) {
		payout := newRepositoryTestPayout(t, "repository-payout")

		if err := payoutRepository.Save(ctx, payout); err != nil {
			t.Fatalf("save payout: %v", err)
		}

		got, err := payoutRepository.GetByID(ctx, payout.ID())
		if err != nil {
			t.Fatalf("get payout by ID: %v", err)
		}
		if got == nil {
			t.Fatal("expected payout, got nil")
		}
		if got.ID() != payout.ID() {
			t.Fatalf("payout ID = %q, want %q", got.ID(), payout.ID())
		}
		if got.Beneficiary().AccountNumber != payout.Beneficiary().AccountNumber {
			t.Fatal("decrypted beneficiary account number does not match")
		}

		byReference, err := payoutRepository.GetByBusinessReference(ctx, payout.BusinessReference())
		if err != nil {
			t.Fatalf("get payout by business reference: %v", err)
		}
		if byReference == nil || byReference.ID() != payout.ID() {
			t.Fatal("payout lookup by business reference returned the wrong payout")
		}

		list, total, err := payoutRepository.ListByStoreID(ctx, payout.StoreID(), 10, 0)
		if err != nil {
			t.Fatalf("list payouts by store ID: %v", err)
		}
		if total != 1 {
			t.Fatalf("total payout count = %d, want 1", total)
		}
		if len(list) != 1 {
			t.Fatalf("payout list length = %d, want 1", len(list))
		}
		if list[0].ID() != payout.ID() {
			t.Fatalf("listed payout ID = %q, want %q", list[0].ID(), payout.ID())
		}
	})

	t.Run("payout attempts", func(t *testing.T) {
		payout := newRepositoryTestPayout(t, "repository-payout-attempt")
		if err := payoutRepository.Save(ctx, payout); err != nil {
			t.Fatalf("save parent payout: %v", err)
		}

		attempt, err := entities.NewPayoutAttempt(payout.ID(), "paystack")
		if err != nil {
			t.Fatalf("create payout attempt: %v", err)
		}

		if err := payoutAttemptRepository.Save(ctx, attempt); err != nil {
			t.Fatalf("save payout attempt: %v", err)
		}

		got, err := payoutAttemptRepository.GetByID(ctx, attempt.ID())
		if err != nil {
			t.Fatalf("get payout attempt by ID: %v", err)
		}
		if got == nil {
			t.Fatal("expected payout attempt, got nil")
		}
		if got.ID() != attempt.ID() {
			t.Fatalf("payout attempt ID = %q, want %q", got.ID(), attempt.ID())
		}
		if got.PayoutID() != payout.ID() {
			t.Fatalf("payout ID = %q, want %q", got.PayoutID(), payout.ID())
		}

		latest, err := payoutAttemptRepository.GetByPayoutID(ctx, payout.ID())
		if err != nil {
			t.Fatalf("get latest payout attempt: %v", err)
		}
		if latest == nil || latest.ID() != attempt.ID() {
			t.Fatal("latest payout attempt lookup returned the wrong attempt")
		}
	})
}

func newRepositoryTestPool(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	_ = godotenv.Load("../../../../../.env")

	databaseURL := os.Getenv("PAYMENT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PAYMENT_TEST_DATABASE_URL is not configured")
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}

	databaseName := strings.TrimPrefix(parsedURL.Path, "/")
	if databaseName == "payment" || !strings.HasSuffix(databaseName, "_test") {
		t.Fatalf("refusing to run integration tests against database %q; use a dedicated *_test database", databaseName)
	}

	ctx := context.Background()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		adminPool.Close()
		t.Fatalf("generate temporary schema name: %v", err)
	}

	schema := "payment_repo_it_" + hex.EncodeToString(randomBytes)
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA "`+schema+`"`); err != nil {
		adminPool.Close()
		t.Fatalf("create temporary integration schema: %v", err)
	}

	appPoolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schema+`" CASCADE`)
		adminPool.Close()
		t.Fatalf("parse application pool config: %v", err)
	}
	appPoolConfig.ConnConfig.RuntimeParams["search_path"] = schema

	appPool, err := pgxpool.NewWithConfig(ctx, appPoolConfig)
	if err != nil {
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schema+`" CASCADE`)
		adminPool.Close()
		t.Fatalf("create isolated application pool: %v", err)
	}

	migrationDB := stdlib.OpenDBFromPool(appPool)
	if err := goose.SetDialect("postgres"); err != nil {
		migrationDB.Close()
		appPool.Close()
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schema+`" CASCADE`)
		adminPool.Close()
		t.Fatalf("set migration dialect: %v", err)
	}
	goose.SetBaseFS(paymentmigrations.FS)

	if err := goose.UpContext(ctx, migrationDB, "."); err != nil {
		migrationDB.Close()
		appPool.Close()
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schema+`" CASCADE`)
		adminPool.Close()
		t.Fatalf("run payment migrations: %v", err)
	}
	if err := migrationDB.Close(); err != nil {
		t.Fatalf("close migration database wrapper: %v", err)
	}

	cleanup := func() {
		appPool.Close()
		if _, err := adminPool.Exec(ctx, `DROP SCHEMA "`+schema+`" CASCADE`); err != nil {
			t.Errorf("drop temporary integration schema: %v", err)
		}
		adminPool.Close()
	}

	return appPool, cleanup
}

func newRepositoryTestPayout(t *testing.T, suffix string) *entities.Payout {
	t.Helper()

	reference, err := value_objects.NewBusinessReference("order", "integration-"+suffix)
	if err != nil {
		t.Fatalf("create payout business reference: %v", err)
	}

	amount, err := value_objects.NewMoney(250000, "NGN")
	if err != nil {
		t.Fatalf("create payout amount: %v", err)
	}

	payout, err := entities.NewPayout(
		reference,
		"00000000-0000-4000-8000-000000000002",
		"00000000-0000-4000-8000-000000000003",
		"vendor settlement",
		amount,
		entities.PayoutBeneficiary{
			AccountName:   "Integration Test Beneficiary",
			AccountNumber: "0123456789",
			BankCode:      "058",
			BankName:      "Test Bank",
			CountryCode:   "NG",
		},
	)
	if err != nil {
		t.Fatalf("create payout: %v", err)
	}

	return payout
}
