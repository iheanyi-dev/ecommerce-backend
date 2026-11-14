package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/iheanyi-dev/ecommerce-backend/services/promotion/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/promotion/domain/valueobjects"
	postgres "github.com/iheanyi-dev/ecommerce-backend/services/promotion/infrastructure/persistence/postgres"
	promotionmigrations "github.com/iheanyi-dev/ecommerce-backend/services/promotion/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func newRepositoryTestPool(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	_ = godotenv.Load("../../../../../.env")

	databaseURL := os.Getenv("PROMOTION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PROMOTION_TEST_DATABASE_URL is not configured")
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}

	databaseName := strings.TrimPrefix(parsedURL.Path, "/")
	if !strings.HasSuffix(databaseName, "_test") {
		t.Fatalf(
			"refusing to run integration tests against database %q; use a dedicated *_test database",
			databaseName,
		)
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

	schema := "promotion_repo_it_" + hex.EncodeToString(randomBytes)
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA "`+schema+`"`); err != nil {
		adminPool.Close()
		t.Fatalf("create temporary integration schema: %v", err)
	}

	dropSchema := func() {
		_, _ = adminPool.Exec(context.Background(), `DROP SCHEMA IF EXISTS "`+schema+`" CASCADE`)
	}

	appPoolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		dropSchema()
		adminPool.Close()
		t.Fatalf("parse application pool config: %v", err)
	}
	appPoolConfig.ConnConfig.RuntimeParams["search_path"] = schema

	appPool, err := pgxpool.NewWithConfig(ctx, appPoolConfig)
	if err != nil {
		dropSchema()
		adminPool.Close()
		t.Fatalf("create isolated application pool: %v", err)
	}

	migrationDB := stdlib.OpenDBFromPool(appPool)

	if err := goose.SetDialect("postgres"); err != nil {
		migrationDB.Close()
		appPool.Close()
		dropSchema()
		adminPool.Close()
		t.Fatalf("set migration dialect: %v", err)
	}
	goose.SetBaseFS(promotionmigrations.FS)

	if err := goose.UpContext(ctx, migrationDB, "."); err != nil {
		migrationDB.Close()
		appPool.Close()
		dropSchema()
		adminPool.Close()
		t.Fatalf("run promotion migrations: %v", err)
	}

	if err := migrationDB.Close(); err != nil {
		t.Fatalf("close migration database wrapper: %v", err)
	}

	cleanup := func() {
		appPool.Close()
		dropSchema()
		adminPool.Close()
	}

	return appPool, cleanup
}

func newTestPromotion(
	t *testing.T,
	storeID string,
	productID, customerID *string,
	now time.Time,
) *entities.Promotion {
	t.Helper()

	percentage := int64(1500)
	discount, err := valueobjects.NewDiscount(&percentage, nil)
	if err != nil {
		t.Fatalf("create promotion discount: %v", err)
	}

	promotion, err := entities.NewPromotion(
		storeID,
		productID,
		customerID,
		discount,
		now.Add(-time.Hour),
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("create promotion: %v", err)
	}

	return promotion
}

func stringPointer(value string) *string {
	return &value
}

func TestPromotionRepositoryPersistenceAndTargeting(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	repository, err := postgres.NewPromotionRepository(pool)
	if err != nil {
		t.Fatalf("create promotion repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	storeID := uuid.NewString()
	productID := uuid.NewString()
	customerID := uuid.NewString()

	storewide := newTestPromotion(t, storeID, nil, nil, now)
	targeted := newTestPromotion(
		t, storeID, stringPointer(productID), stringPointer(customerID), now,
	)
	productOnly := newTestPromotion(
		t, storeID, stringPointer(uuid.NewString()), nil, now,
	)

	for _, promotion := range []*entities.Promotion{
		storewide,
		targeted,
		productOnly,
	} {
		if err := repository.CreateIfNoUnexpiredConflict(ctx, promotion, now); err != nil {
			t.Fatalf("create promotion %s: %v", promotion.ID(), err)
		}
	}

	got, err := repository.GetByID(ctx, targeted.ID())
	if err != nil {
		t.Fatalf("get promotion: %v", err)
	}
	if got == nil {
		t.Fatal("expected persisted promotion")
	}
	if got.ID() != targeted.ID() || got.StoreID() != storeID {
		t.Fatalf("unexpected reconstituted promotion: ID=%s StoreID=%s", got.ID(), got.StoreID())
	}
	if got.ProductID() == nil || *got.ProductID() != productID {
		t.Fatal("product targeting was not reconstituted")
	}
	if got.CustomerID() == nil || *got.CustomerID() != customerID {
		t.Fatal("customer targeting was not reconstituted")
	}

	missing, err := repository.GetByID(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("get missing promotion: %v", err)
	}
	if missing != nil {
		t.Fatal("expected nil for missing promotion")
	}

	byStore, err := repository.ListByStore(ctx, storeID, 10, 0)
	if err != nil {
		t.Fatalf("list promotions by store: %v", err)
	}
	if len(byStore) != 3 {
		t.Fatalf("expected 3 store promotions, got %d", len(byStore))
	}

	active, err := repository.ListActiveForTarget(
		ctx, storeID, productID, customerID, now,
	)
	if err != nil {
		t.Fatalf("list active promotions for target: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected storewide and matching targeted promotions, got %d", len(active))
	}

	available, err := repository.ListAvailableForCustomer(
		ctx, customerID, now, 10, 0,
	)
	if err != nil {
		t.Fatalf("list available promotions: %v", err)
	}
	if len(available) != 2 {
		t.Fatalf("expected storewide and customer-targeted promotions, got %d", len(available))
	}

	if err := repository.Delete(ctx, productOnly.ID()); err != nil {
		t.Fatalf("delete promotion: %v", err)
	}
	deleted, err := repository.GetByID(ctx, productOnly.ID())
	if err != nil {
		t.Fatalf("get deleted promotion: %v", err)
	}
	if deleted != nil {
		t.Fatal("expected deleted promotion to be absent")
	}
}

func TestPromotionRepositoryRejectsUnexpiredTargetConflict(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	repository, err := postgres.NewPromotionRepository(pool)
	if err != nil {
		t.Fatalf("create promotion repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	storeID := uuid.NewString()
	productID := uuid.NewString()
	customerID := uuid.NewString()

	first := newTestPromotion(
		t, storeID, stringPointer(productID), stringPointer(customerID), now,
	)
	second := newTestPromotion(
		t, storeID, stringPointer(productID), stringPointer(customerID), now,
	)

	if err := repository.CreateIfNoUnexpiredConflict(ctx, first, now); err != nil {
		t.Fatalf("create first promotion: %v", err)
	}

	err = repository.CreateIfNoUnexpiredConflict(ctx, second, now)
	if !errors.Is(err, apperrors.ErrPromotionConflict) {
		t.Fatalf("expected ErrPromotionConflict, got %v", err)
	}
}

func TestPromotionOutboxRepositoryLifecycle(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	repository, err := postgres.NewPromotionOutboxRepository(pool)
	if err != nil {
		t.Fatalf("create outbox repository: %v", err)
	}

	ctx := context.Background()
	eventID := uuid.NewString()
	promotionID := uuid.NewString()

	payload, err := json.Marshal(map[string]string{
		"promo_id":   promotionID,
		"store_id":   uuid.NewString(),
		"product_id": uuid.NewString(),
		"status":     "cancelled",
	})
	if err != nil {
		t.Fatalf("marshal event payload: %v", err)
	}

	if err := repository.Create(
		ctx, eventID, promotionID, "promotion.cancelled", payload,
	); err != nil {
		t.Fatalf("create outbox event: %v", err)
	}

	pending, err := repository.ClaimPending(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("claim pending events: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 claimed event, got %d", len(pending))
	}

	event := pending[0]
	if event.EventID != eventID {
		t.Fatalf("unexpected event ID: %s", event.EventID)
	}
	if event.EventType != "promotion.cancelled" {
		t.Fatalf("unexpected event type: %s", event.EventType)
	}
	if event.ClaimToken == "" {
		t.Fatal("expected a claim token")
	}

	var gotPayload map[string]string
	if err := json.Unmarshal(event.Payload, &gotPayload); err != nil {
		t.Fatalf("decode persisted event payload: %v", err)
	}
	if gotPayload["promo_id"] != promotionID ||
		gotPayload["status"] != "cancelled" {
		t.Fatalf("unexpected event payload: %+v", gotPayload)
	}
	if event.Attempts != 0 {
		t.Fatalf("expected 0 attempts, got %d", event.Attempts)
	}

	// A currently claimed event must not be claimable by another worker.
	claimedAgain, err := repository.ClaimPending(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("claim events from second worker: %v", err)
	}
	if len(claimedAgain) != 0 {
		t.Fatalf("expected no events claimed by second worker, got %d", len(claimedAgain))
	}

	if err := repository.RecordFailure(
		ctx, eventID, event.ClaimToken, fmt.Errorf("broker unavailable"),
	); err != nil {
		t.Fatalf("record event failure: %v", err)
	}

	pending, err = repository.ClaimPending(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("reclaim pending events after failure: %v", err)
	}
	if len(pending) != 1 || pending[0].Attempts != 1 {
		t.Fatalf("expected one event with one attempt, got %+v", pending)
	}

	reclaimed := pending[0]
	if reclaimed.ClaimToken == event.ClaimToken {
		t.Fatal("expected a new claim token after failure")
	}

	// A stale worker must not be able to acknowledge the reclaimed event.
	if err := repository.MarkPublished(ctx, eventID, event.ClaimToken); err == nil {
		t.Fatal("expected stale claim token to be rejected")
	}

	if err := repository.MarkPublished(ctx, eventID, reclaimed.ClaimToken); err != nil {
		t.Fatalf("mark event published: %v", err)
	}

	pending, err = repository.ClaimPending(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("claim events after publication: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending events, got %d", len(pending))
	}
}

func TestPromotionUnitOfWorkRollsBackPromotionAndOutbox(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	promotionRepository, err := postgres.NewPromotionRepository(pool)
	if err != nil {
		t.Fatalf("create promotion repository: %v", err)
	}
	outboxRepository, err := postgres.NewPromotionOutboxRepository(pool)
	if err != nil {
		t.Fatalf("create outbox repository: %v", err)
	}
	uow, err := postgres.NewUnitOfWork(pool)
	if err != nil {
		t.Fatalf("create unit of work: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	promotion := newTestPromotion(t, uuid.NewString(), nil, nil, now)
	eventID := uuid.NewString()

	err = uow.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := promotionRepository.CreateIfNoUnexpiredConflict(
			txCtx, promotion, now,
		); err != nil {
			return err
		}

		if err := outboxRepository.Create(
			txCtx, eventID, promotion.ID(), "promotion.created", []byte(`{"status":"active"}`),
		); err != nil {
			return err
		}

		// Duplicate event ID forces a database constraint violation.
		return outboxRepository.Create(
			txCtx, eventID, promotion.ID(), "promotion.activated", []byte(`{"status":"active"}`),
		)
	})
	if err == nil {
		t.Fatal("expected transaction to fail")
	}

	got, err := promotionRepository.GetByID(ctx, promotion.ID())
	if err != nil {
		t.Fatalf("get promotion after rollback: %v", err)
	}
	if got != nil {
		t.Fatal("promotion should have been rolled back")
	}

	pending, err := outboxRepository.ClaimPending(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("get outbox after rollback: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("outbox events should have been rolled back, got %d", len(pending))
	}
}

func TestPromotionRepositoryConcurrentTargetConflict(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	repository, err := postgres.NewPromotionRepository(pool)
	if err != nil {
		t.Fatalf("create promotion repository: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	storeID := uuid.NewString()
	productID := uuid.NewString()
	customerID := uuid.NewString()

	const workers = 8
	start := make(chan struct{})
	results := make(chan error, workers)

	for i := 0; i < workers; i++ {
		promotion := newTestPromotion(
			t,
			storeID,
			stringPointer(productID),
			stringPointer(customerID),
			now,
		)

		go func() {
			<-start
			results <- repository.CreateIfNoUnexpiredConflict(
				context.Background(),
				promotion,
				now,
			)
		}()
	}

	close(start)

	created := 0
	conflicts := 0

	for i := 0; i < workers; i++ {
		err := <-results
		switch {
		case err == nil:
			created++
		case errors.Is(err, apperrors.ErrPromotionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent creation error: %v", err)
		}
	}

	if created != 1 || conflicts != workers-1 {
		t.Fatalf(
			"expected 1 creation and %d conflicts; got %d creations and %d conflicts",
			workers-1,
			created,
			conflicts,
		)
	}
}

func TestPromotionRepositoryListsScheduledPromotionsDue(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	repository, err := postgres.NewPromotionRepository(pool)
	if err != nil {
		t.Fatalf("create promotion repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	create := func(startsAt, endsAt, createdAt time.Time) *entities.Promotion {
		t.Helper()

		percentage := int64(1500)
		discount, err := valueobjects.NewDiscount(&percentage, nil)
		if err != nil {
			t.Fatalf("create discount: %v", err)
		}

		promotion, err := entities.NewPromotion(
			uuid.NewString(), nil, nil, discount, startsAt, endsAt,
		)
		if err != nil {
			t.Fatalf("create promotion: %v", err)
		}
		if err := repository.CreateIfNoUnexpiredConflict(ctx, promotion, createdAt); err != nil {
			t.Fatalf("persist promotion: %v", err)
		}
		return promotion
	}

	first := create(now.Add(-3*time.Minute), now.Add(time.Hour), now.Add(-4*time.Minute))
	second := create(now.Add(-2*time.Minute), now.Add(time.Hour), now.Add(-3*time.Minute))
	future := create(now.Add(time.Hour), now.Add(2*time.Hour), now)

	got, err := repository.ListScheduledDue(ctx, now, 10)
	if err != nil {
		t.Fatalf("list scheduled promotions due: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 scheduled promotions, got %d: %v", len(got), got)
	}
	if got[0] != first.ID() || got[1] != second.ID() {
		t.Fatalf("unexpected scheduled promotion ordering: %v", got)
	}
	for _, id := range got {
		if id == future.ID() {
			t.Fatal("future promotion was returned as due")
		}
	}
}

func TestPromotionRepositoryListsExpiredPromotionsDue(t *testing.T) {
	pool, cleanup := newRepositoryTestPool(t)
	defer cleanup()

	repository, err := postgres.NewPromotionRepository(pool)
	if err != nil {
		t.Fatalf("create promotion repository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	create := func(startsAt, endsAt, createdAt time.Time) *entities.Promotion {
		t.Helper()

		percentage := int64(1500)
		discount, err := valueobjects.NewDiscount(&percentage, nil)
		if err != nil {
			t.Fatalf("create discount: %v", err)
		}

		promotion, err := entities.NewPromotion(
			uuid.NewString(), nil, nil, discount, startsAt, endsAt,
		)
		if err != nil {
			t.Fatalf("create promotion: %v", err)
		}
		if err := repository.CreateIfNoUnexpiredConflict(ctx, promotion, createdAt); err != nil {
			t.Fatalf("persist promotion: %v", err)
		}
		return promotion
	}

	// Both promotions are persisted before their end times.
	active := create(now.Add(-3*time.Hour), now.Add(-2*time.Minute), now.Add(-time.Hour))
	scheduled := create(now.Add(-2*time.Hour), now.Add(-time.Minute), now.Add(-3*time.Hour))

	// This promotion is still active and must not be returned.
	notExpired := create(now.Add(-time.Hour), now.Add(time.Hour), now.Add(-30*time.Minute))

	got, err := repository.ListExpiredDue(ctx, now, 10)
	if err != nil {
		t.Fatalf("list expired promotions due: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 expired candidates, got %d: %v", len(got), got)
	}
	if got[0] != active.ID() || got[1] != scheduled.ID() {
		t.Fatalf("unexpected expired promotion ordering: %v", got)
	}
	for _, id := range got {
		if id == notExpired.ID() {
			t.Fatal("unexpired promotion was returned")
		}
	}
}
