package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres"
)

// newRepository creates a real PostgreSQL-backed Store repository for an
// integration test.
func newRepository(t *testing.T) (*postgres.StoreRepository, *TestDatabase) {
	t.Helper()

	db := NewTestDatabase(t)

	return postgres.NewStoreRepository(
		postgres.NewQueries(db.Pool),
	), db
}

// newTestStore creates a valid Store aggregate with unique owner and slug
// values so tests do not collide with one another.
func newTestStore(t *testing.T, name string) *entities.Store {
	t.Helper()

	store, err := entities.NewStore(
		uuid.New(),
		name,
		"store-"+uuid.NewString(),
		"Integration test store.",
		nil,
		"basic",
	)
	if err != nil {
		t.Fatalf("create test store entity: %v", err)
	}

	return store
}

// persistStore persists a test aggregate and registers automatic cleanup.
func persistStore(
	t *testing.T,
	repository *postgres.StoreRepository,
	store *entities.Store,
) {
	t.Helper()

	ctx := context.Background()

	if err := repository.Create(ctx, store); err != nil {
		t.Fatalf("create store: %v", err)
	}

	t.Cleanup(func() {
		if err := repository.Delete(ctx, store.ID().Value()); err != nil {
			t.Errorf("cleanup store %s: %v", store.ID().Value(), err)
		}
	})
}

// TestStoreRepository_Lookups verifies the repository's supported lookup
// operations against the real PostgreSQL database.
func TestStoreRepository_Lookups(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	store := newTestStore(t, "Lookup Store")
	persistStore(t, repository, store)

	t.Run("find by owner ID", func(t *testing.T) {
		got, err := repository.FindByOwnerID(ctx, store.OwnerID().Value())
		if err != nil {
			t.Fatalf("find store by owner ID: %v", err)
		}

		if got == nil {
			t.Fatal("expected store, got nil")
		}

		if got.ID().Value() != store.ID().Value() {
			t.Fatalf(
				"store ID mismatch: got %s, want %s",
				got.ID().Value(),
				store.ID().Value(),
			)
		}
	})

	t.Run("find by slug", func(t *testing.T) {
		got, err := repository.FindBySlug(ctx, store.Slug().Value())
		if err != nil {
			t.Fatalf("find store by slug: %v", err)
		}

		if got == nil {
			t.Fatal("expected store, got nil")
		}

		if got.ID().Value() != store.ID().Value() {
			t.Fatalf(
				"store ID mismatch: got %s, want %s",
				got.ID().Value(),
				store.ID().Value(),
			)
		}
	})

	t.Run("slug exists", func(t *testing.T) {
		exists, err := repository.ExistsBySlug(ctx, store.Slug().Value())
		if err != nil {
			t.Fatalf("check existing slug: %v", err)
		}

		if !exists {
			t.Fatal("expected slug to exist")
		}
	})

	t.Run("missing slug does not exist", func(t *testing.T) {
		exists, err := repository.ExistsBySlug(
			ctx,
			"missing-"+uuid.NewString(),
		)
		if err != nil {
			t.Fatalf("check missing slug: %v", err)
		}

		if exists {
			t.Fatal("expected missing slug not to exist")
		}
	})

	t.Run("missing ID returns not found", func(t *testing.T) {
		_, err := repository.FindByID(ctx, uuid.New())

		if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
			t.Fatalf(
				"expected ErrStoreNotFound, got %v",
				err,
			)
		}
	})

	t.Run("missing owner returns not found", func(t *testing.T) {
		_, err := repository.FindByOwnerID(ctx, uuid.New())

		if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
			t.Fatalf(
				"expected ErrStoreNotFound, got %v",
				err,
			)
		}
	})

	t.Run("missing slug returns not found", func(t *testing.T) {
		_, err := repository.FindBySlug(
			ctx,
			"missing-"+uuid.NewString(),
		)

		if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
			t.Fatalf(
				"expected ErrStoreNotFound, got %v",
				err,
			)
		}
	})
}

// TestStoreRepository_Uniqueness verifies that PostgreSQL uniqueness
// constraints are translated into the repository's application errors.
func TestStoreRepository_Uniqueness(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	original := newTestStore(t, "Original Store")
	persistStore(t, repository, original)

	t.Run("duplicate owner", func(t *testing.T) {
		duplicate, err := entities.NewStore(
			original.OwnerID().Value(),
			"Second Store",
			"second-"+uuid.NewString(),
			"Duplicate owner integration test.",
			nil,
			"basic",
		)
		if err != nil {
			t.Fatalf("create duplicate-owner entity: %v", err)
		}

		err = repository.Create(ctx, duplicate)

		if !errors.Is(err, applicationerrors.ErrStoreAlreadyExists) {
			t.Fatalf(
				"expected ErrStoreAlreadyExists, got %v",
				err,
			)
		}
	})

	t.Run("duplicate slug", func(t *testing.T) {
		duplicate, err := entities.NewStore(
			uuid.New(),
			"Second Store",
			original.Slug().Value(),
			"Duplicate slug integration test.",
			nil,
			"basic",
		)
		if err != nil {
			t.Fatalf("create duplicate-slug entity: %v", err)
		}

		err = repository.Create(ctx, duplicate)

		if !errors.Is(err, applicationerrors.ErrStoreSlugAlreadyExists) {
			t.Fatalf(
				"expected ErrStoreSlugAlreadyExists, got %v",
				err,
			)
		}
	})
}

// TestStoreRepository_Update verifies that Update persists the complete
// mutable Store aggregate, including image reference and status/plan values.
func TestStoreRepository_Update(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	store := newTestStore(t, "Original Name")
	persistStore(t, repository, store)

	if err := store.ChangeName("Updated Store Name"); err != nil {
		t.Fatalf("change name: %v", err)
	}

	if err := store.ChangeSlug("updated-" + uuid.NewString()); err != nil {
		t.Fatalf("change slug: %v", err)
	}

	if err := store.ChangeDescription("Updated integration-test description."); err != nil {
		t.Fatalf("change description: %v", err)
	}

	store.SetImageReference(
		"stores/" + store.ID().Value().String() + ".webp",
	)

	if err := repository.Update(ctx, store); err != nil {
		t.Fatalf("update store: %v", err)
	}

	got, err := repository.FindByID(ctx, store.ID().Value())
	if err != nil {
		t.Fatalf("find updated store: %v", err)
	}

	if got.Name().Value() != "Updated Store Name" {
		t.Fatalf(
			"name mismatch: got %q, want %q",
			got.Name().Value(),
			"Updated Store Name",
		)
	}

	if got.Slug().Value() != store.Slug().Value() {
		t.Fatalf(
			"slug mismatch: got %q, want %q",
			got.Slug().Value(),
			store.Slug().Value(),
		)
	}

	if got.Description().Value() != "Updated integration-test description." {
		t.Fatalf(
			"description mismatch: got %q, want %q",
			got.Description().Value(),
			"Updated integration-test description.",
		)
	}

	if got.ImageReference() == nil {
		t.Fatal("expected image reference, got nil")
	}

	if *got.ImageReference() != *store.ImageReference() {
		t.Fatalf(
			"image reference mismatch: got %q, want %q",
			*got.ImageReference(),
			*store.ImageReference(),
		)
	}
}

// TestStoreRepository_ChangePlan verifies that a plan transition persists the
// complete aggregate in one repository operation.
func TestStoreRepository_ChangePlan(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	store := newTestStore(t, "Plan Store")
	persistStore(t, repository, store)

	// Reconstitute the aggregate with the target plan/status so this test
	// exercises the repository contract without duplicating the domain's
	// ChangePlan transition logic.
	changed, err := entities.ReconstituteStore(
		store.ID().Value(),
		store.OwnerID().Value(),
		store.Name().Value(),
		store.Slug().Value(),
		store.Description().Value(),
		store.ImageReference(),
		"premium",
		"inactive",
		store.CreatedAt(),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("reconstitute target plan state: %v", err)
	}

	if err := repository.ChangePlan(ctx, changed); err != nil {
		t.Fatalf("change store plan: %v", err)
	}

	got, err := repository.FindByID(ctx, store.ID().Value())
	if err != nil {
		t.Fatalf("find store after plan change: %v", err)
	}

	if got.Plan().Type() != changed.Plan().Type() {
		t.Fatalf(
			"plan mismatch: got %q, want %q",
			got.Plan().Type(),
			changed.Plan().Type(),
		)
	}

	if got.Status() != changed.Status() {
		t.Fatalf(
			"status mismatch: got %q, want %q",
			got.Status(),
			changed.Status(),
		)
	}
}

// TestStoreRepository_ChangeStatus verifies that status persistence is
// performed independently through the ChangeStatus repository contract.
func TestStoreRepository_ChangeStatus(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	store := newTestStore(t, "Status Store")
	persistStore(t, repository, store)

	originalUpdatedAt := store.UpdatedAt()

	// Give PostgreSQL enough time to produce a distinct NOW() value.
	time.Sleep(time.Millisecond)

	if err := repository.ChangeStatus(
		ctx,
		store.ID().Value(),
		"inactive",
	); err != nil {
		t.Fatalf("change store status: %v", err)
	}

	got, err := repository.FindByID(ctx, store.ID().Value())
	if err != nil {
		t.Fatalf("find store after status change: %v", err)
	}

	if string(got.Status()) != "inactive" {
		t.Fatalf(
			"status mismatch: got %q, want %q",
			got.Status(),
			"inactive",
		)
	}

	if !got.UpdatedAt().After(originalUpdatedAt) {
		t.Fatalf(
			"expected updated_at to advance: got %s, original %s",
			got.UpdatedAt(),
			originalUpdatedAt,
		)
	}
}

// TestStoreRepository_ListActive verifies database-side active filtering,
// search, pagination, and total-count behavior.
func TestStoreRepository_ListActive(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	activeAlpha := newTestStore(t, "Alpha Electronics")
	persistStore(t, repository, activeAlpha)

	activeBeta := newTestStore(t, "Beta Electronics")
	persistStore(t, repository, activeBeta)

	inactive := newTestStore(t, "Inactive Electronics")
	persistStore(t, repository, inactive)

	if err := inactive.Deactivate(); err != nil {
		t.Fatalf("deactivate test store: %v", err)
	}

	if err := repository.Update(ctx, inactive); err != nil {
		t.Fatalf("persist inactive store: %v", err)
	}

	t.Run("active stores only", func(t *testing.T) {
		stores, total, err := repository.ListActive(ctx, "", 1, 100)
		if err != nil {
			t.Fatalf("list active stores: %v", err)
		}

		if total < 2 {
			t.Fatalf("expected at least 2 active stores, got %d", total)
		}

		for _, store := range stores {
			if string(store.Status()) != "active" {
				t.Fatalf(
					"list returned non-active store %s with status %q",
					store.ID().Value(),
					store.Status(),
				)
			}
		}
	})

	t.Run("search by name", func(t *testing.T) {
		stores, total, err := repository.ListActive(
			ctx,
			"Alpha Electronics",
			1,
			100,
		)
		if err != nil {
			t.Fatalf("search active stores: %v", err)
		}

		if total != 1 {
			t.Fatalf("expected 1 matching store, got %d", total)
		}

		if len(stores) != 1 {
			t.Fatalf("expected 1 returned store, got %d", len(stores))
		}

		if stores[0].ID().Value() != activeAlpha.ID().Value() {
			t.Fatalf(
				"returned wrong store: got %s, want %s",
				stores[0].ID().Value(),
				activeAlpha.ID().Value(),
			)
		}
	})

	t.Run("search by slug", func(t *testing.T) {
		stores, total, err := repository.ListActive(
			ctx,
			activeBeta.Slug().Value(),
			1,
			100,
		)
		if err != nil {
			t.Fatalf("search active stores by slug: %v", err)
		}

		if total != 1 {
			t.Fatalf("expected 1 matching store, got %d", total)
		}

		if len(stores) != 1 {
			t.Fatalf("expected 1 returned store, got %d", len(stores))
		}

		if stores[0].ID().Value() != activeBeta.ID().Value() {
			t.Fatalf(
				"returned wrong store: got %s, want %s",
				stores[0].ID().Value(),
				activeBeta.ID().Value(),
			)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		stores, total, err := repository.ListActive(ctx, "", 1, 1)
		if err != nil {
			t.Fatalf("list first page: %v", err)
		}

		if total < 2 {
			t.Fatalf("expected at least 2 active stores, got %d", total)
		}

		if len(stores) != 1 {
			t.Fatalf("expected 1 store on first page, got %d", len(stores))
		}

		pageTwo, pageTwoTotal, err := repository.ListActive(
			ctx,
			"",
			2,
			1,
		)
		if err != nil {
			t.Fatalf("list second page: %v", err)
		}

		if pageTwoTotal != total {
			t.Fatalf(
				"total mismatch between pages: got %d, want %d",
				pageTwoTotal,
				total,
			)
		}

		if len(pageTwo) != 1 {
			t.Fatalf("expected 1 store on second page, got %d", len(pageTwo))
		}

		if pageTwo[0].ID().Value() == stores[0].ID().Value() {
			t.Fatal("pagination returned the same store on both pages")
		}
	})
}

// TestStoreRepository_ListActiveRejectsInvalidPagination verifies that invalid
// pagination never reaches PostgreSQL.
func TestStoreRepository_ListActiveRejectsInvalidPagination(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		page     int
		pageSize int
	}{
		{
			name:     "zero page",
			page:     0,
			pageSize: 10,
		},
		{
			name:     "negative page",
			page:     -1,
			pageSize: 10,
		},
		{
			name:     "zero page size",
			page:     1,
			pageSize: 0,
		},
		{
			name:     "negative page size",
			page:     1,
			pageSize: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := repository.ListActive(
				ctx,
				"",
				tt.page,
				tt.pageSize,
			)

			if !errors.Is(err, applicationerrors.ErrInvalidPagination) {
				t.Fatalf(
					"expected ErrInvalidPagination, got %v",
					err,
				)
			}
		})
	}
}

// TestStoreRepository_Delete verifies deletion and confirms that a deleted
// aggregate can no longer be retrieved.
func TestStoreRepository_Delete(t *testing.T) {
	repository, _ := newRepository(t)
	ctx := context.Background()

	store := newTestStore(t, "Delete Store")

	if err := repository.Create(ctx, store); err != nil {
		t.Fatalf("create store: %v", err)
	}

	if err := repository.Delete(ctx, store.ID().Value()); err != nil {
		t.Fatalf("delete store: %v", err)
	}

	_, err := repository.FindByID(ctx, store.ID().Value())

	if !errors.Is(err, applicationerrors.ErrStoreNotFound) {
		t.Fatalf(
			"expected ErrStoreNotFound after deletion, got %v",
			err,
		)
	}
}
