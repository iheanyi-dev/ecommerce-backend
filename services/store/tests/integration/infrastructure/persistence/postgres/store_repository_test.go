package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres"
)

// TestStoreRepository_CreateAndFindByID verifies the complete persistence
// round-trip for a Store aggregate.
//
// This is intentionally an integration test rather than a mocked repository
// test: it proves that the real PostgreSQL schema, SQLC-generated queries,
// database type conversions, and domain reconstitution work together.
func TestStoreRepository_CreateAndFindByID(t *testing.T) {
	db := NewTestDatabase(t)

	repository := postgres.NewStoreRepository(
		// The repository receives SQLC's generated query object rather than
		// talking to pgx directly. This keeps the repository dependent on the
		// generated persistence abstraction.
		postgres.NewQueries(db.Pool),
	)

	ownerID := uuid.New()

	store, err := entities.NewStore(
		ownerID,
		"Integration Store",
		"integration-store-"+uuid.NewString()[:8],
		"Store created by the PostgreSQL integration test.",
		nil,
		"basic",
	)
	if err != nil {
		t.Fatalf("create test store entity: %v", err)
	}

	ctx := context.Background()

	if err := repository.Create(ctx, store); err != nil {
		t.Fatalf("create store through repository: %v", err)
	}

	t.Cleanup(func() {
		if err := repository.Delete(ctx, store.ID().Value()); err != nil {
			t.Errorf("cleanup store: %v", err)
		}
	})

	got, err := repository.FindByID(ctx, store.ID().Value())
	if err != nil {
		t.Fatalf("find store by ID: %v", err)
	}

	if got == nil {
		t.Fatal("expected store, got nil")
	}

	if got.ID().Value() != store.ID().Value() {
		t.Fatalf("store ID mismatch: got %s, want %s", got.ID().Value(), store.ID().Value())
	}

	if got.OwnerID().Value() != store.OwnerID().Value() {
		t.Fatalf(
			"owner ID mismatch: got %s, want %s",
			got.OwnerID().Value(),
			store.OwnerID().Value(),
		)
	}

	if got.Name().Value() != store.Name().Value() {
		t.Fatalf(
			"name mismatch: got %q, want %q",
			got.Name().Value(),
			store.Name().Value(),
		)
	}

	if got.Slug().Value() != store.Slug().Value() {
		t.Fatalf(
			"slug mismatch: got %q, want %q",
			got.Slug().Value(),
			store.Slug().Value(),
		)
	}

	if got.Description().Value() != store.Description().Value() {
		t.Fatalf(
			"description mismatch: got %q, want %q",
			got.Description().Value(),
			store.Description().Value(),
		)
	}

	if got.ImageReference() != nil {
		t.Fatalf("expected nil image reference, got %q", *got.ImageReference())
	}

	if got.Status() != store.Status() {
		t.Fatalf(
			"status mismatch: got %q, want %q",
			got.Status(),
			store.Status(),
		)
	}

	if got.Plan().Type() != store.Plan().Type() {
		t.Fatalf(
			"plan mismatch: got %q, want %q",
			got.Plan().Type(),
			store.Plan().Type(),
		)
	}

	// PostgreSQL TIMESTAMPTZ stores timestamps with microsecond precision.
	// Go's time.Now() can contain nanoseconds, so normalize the original
	// aggregate timestamps to PostgreSQL's precision before comparing them.
	expectedCreatedAt := store.CreatedAt().Truncate(time.Microsecond)

	if !got.CreatedAt().Equal(expectedCreatedAt) {
		t.Fatalf(
			"created_at mismatch: got %s, want %s",
			got.CreatedAt(),
			expectedCreatedAt,
		)
	}

	expectedUpdatedAt := store.UpdatedAt().Truncate(time.Microsecond)

	if !got.UpdatedAt().Equal(expectedUpdatedAt) {
		t.Fatalf(
			"updated_at mismatch: got %s, want %s",
			got.UpdatedAt(),
			expectedUpdatedAt,
		)
	}
}
