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

	applicationerrors "github.com/iheanyi-dev/ecommerce-backend/services/store/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/store/domain/entities"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres/generated"
)

// StoreRepository is the PostgreSQL implementation of the StoreRepository
// application port.
//
// Persistence translation belongs here. Store business rules remain in the
// domain aggregate and application use cases.
type StoreRepository struct {
	queries *generated.Queries
}

// NewStoreRepository creates a PostgreSQL-backed Store repository.
func NewStoreRepository(queries *generated.Queries) *StoreRepository {
	return &StoreRepository{queries: queries}
}

// Compile-time contract verification.
var _ ports.StoreRepository = (*StoreRepository)(nil)

// Create persists a newly created Store aggregate.
func (r *StoreRepository) Create(
	ctx context.Context,
	store *entities.Store,
) error {
	if store == nil {
		return errors.New("store cannot be nil")
	}

	err := r.queries.CreateStore(
		ctx,
		generated.CreateStoreParams{
			ID:             toUUID(store.ID().Value()),
			OwnerID:        toUUID(store.OwnerID().Value()),
			Name:           store.Name().Value(),
			Slug:           store.Slug().Value(),
			Description:    store.Description().Value(),
			ImageReference: toNullableText(store.ImageReference()),
			Status:         string(store.Status()),
			Plan:           string(store.Plan().Type()),
			CreatedAt:      toTimestamp(store.CreatedAt()),
			UpdatedAt:      toTimestamp(store.UpdatedAt()),
		},
	)
	if err != nil {
		return translateWriteError("create store", err)
	}

	return nil
}

// Delete removes a Store by ID.
//
// The application layer uses this as a compensating action when Store
// creation succeeded but the synchronous Identity update subsequently fails.
func (r *StoreRepository) Delete(
	ctx context.Context,
	storeID uuid.UUID,
) error {
	err := r.queries.DeleteStore(
		ctx,
		toUUID(storeID),
	)
	if err != nil {
		return fmt.Errorf(
			"delete store %s: %w",
			storeID,
			err,
		)
	}

	return nil
}

// FindByID retrieves a Store by its identifier.
func (r *StoreRepository) FindByID(
	ctx context.Context,
	storeID uuid.UUID,
) (*entities.Store, error) {
	row, err := r.queries.GetStoreByID(
		ctx,
		toUUID(storeID),
	)
	if err != nil {
		return nil, translateReadError("find store by ID", err)
	}

	return fromRow(row)
}

// FindByOwnerID retrieves the Store owned by a user.
func (r *StoreRepository) FindByOwnerID(
	ctx context.Context,
	ownerID uuid.UUID,
) (*entities.Store, error) {
	row, err := r.queries.GetStoreByOwnerID(
		ctx,
		toUUID(ownerID),
	)
	if err != nil {
		return nil, translateReadError("find store by owner ID", err)
	}

	return fromRow(row)
}

// FindBySlug retrieves a Store by its public slug.
func (r *StoreRepository) FindBySlug(
	ctx context.Context,
	slug string,
) (*entities.Store, error) {
	row, err := r.queries.GetStoreBySlug(
		ctx,
		slug,
	)
	if err != nil {
		return nil, translateReadError("find store by slug", err)
	}

	return fromRow(row)
}

// ExistsBySlug reports whether a Store already uses the supplied slug.
func (r *StoreRepository) ExistsBySlug(
	ctx context.Context,
	slug string,
) (bool, error) {
	exists, err := r.queries.StoreExistsBySlug(
		ctx,
		slug,
	)
	if err != nil {
		return false, fmt.Errorf(
			"check store slug existence: %w",
			err,
		)
	}

	return exists, nil
}

// ListActive retrieves active Stores using database-side filtering and
// pagination.
func (r *StoreRepository) ListActive(
	ctx context.Context,
	query string,
	page int,
	pageSize int,
) ([]*entities.Store, int, error) {
	if page < 1 || pageSize < 1 {
		return nil, 0, applicationerrors.ErrInvalidPagination
	}

	if pageSize > math.MaxInt32 {
		return nil, 0, applicationerrors.ErrInvalidPagination
	}

	offset64 := int64(page-1) * int64(pageSize)

	if offset64 > math.MaxInt32 {
		return nil, 0, applicationerrors.ErrInvalidPagination
	}

	rows, err := r.queries.ListActiveStores(
		ctx,
		generated.ListActiveStoresParams{
			Column1: query,
			Limit:   int32(pageSize),
			Offset:  int32(offset64),
		},
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"list active stores: %w",
			err,
		)
	}

	total, err := r.queries.CountActiveStores(
		ctx,
		query,
	)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"count active stores: %w",
			err,
		)
	}

	stores := make([]*entities.Store, 0, len(rows))

	for _, row := range rows {
		store, err := fromRow(row)
		if err != nil {
			return nil, 0, fmt.Errorf(
				"reconstitute listed store: %w",
				err,
			)
		}

		stores = append(stores, store)
	}

	return stores, int(total), nil
}

// Update persists the complete Store aggregate.
func (r *StoreRepository) Update(
	ctx context.Context,
	store *entities.Store,
) error {
	if store == nil {
		return errors.New("store cannot be nil")
	}

	err := r.queries.UpdateStore(
		ctx,
		generated.UpdateStoreParams{
			ID:             toUUID(store.ID().Value()),
			Name:           store.Name().Value(),
			Slug:           store.Slug().Value(),
			Description:    store.Description().Value(),
			ImageReference: toNullableText(store.ImageReference()),
			Status:         string(store.Status()),
			Plan:           string(store.Plan().Type()),
			UpdatedAt:      toTimestamp(store.UpdatedAt()),
		},
	)
	if err != nil {
		return translateWriteError("update store", err)
	}

	return nil
}

// ChangePlan persists the complete aggregate after a domain plan transition.
//
// Plan changes may also change Store status, so both values are persisted in
// one repository operation. We intentionally do not call ChangeStatus here.
func (r *StoreRepository) ChangePlan(
	ctx context.Context,
	store *entities.Store,
) error {
	if store == nil {
		return errors.New("store cannot be nil")
	}

	err := r.queries.ChangeStorePlan(
		ctx,
		generated.ChangeStorePlanParams{
			ID:             toUUID(store.ID().Value()),
			Name:           store.Name().Value(),
			Slug:           store.Slug().Value(),
			Description:    store.Description().Value(),
			ImageReference: toNullableText(store.ImageReference()),
			Status:         string(store.Status()),
			Plan:           string(store.Plan().Type()),
			UpdatedAt:      toTimestamp(store.UpdatedAt()),
		},
	)
	if err != nil {
		return translateWriteError("change store plan", err)
	}

	return nil
}

// ChangeStatus persists an independently validated status transition.
//
// The repository contract intentionally accepts only Store ID and status.
// PostgreSQL therefore owns updated_at for this operation through NOW().
func (r *StoreRepository) ChangeStatus(
	ctx context.Context,
	storeID uuid.UUID,
	status string,
) error {
	err := r.queries.ChangeStoreStatus(
		ctx,
		generated.ChangeStoreStatusParams{
			ID:     toUUID(storeID),
			Status: status,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"change store status: %w",
			err,
		)
	}

	return nil
}

// fromRow reconstructs the domain aggregate from the SQLC model.
//
// Reconstitution goes through the domain factory rather than constructing
// Store directly, ensuring persisted values still pass domain validation.
func fromRow(row generated.Store) (*entities.Store, error) {
	storeID, err := fromUUID(row.ID)
	if err != nil {
		return nil, fmt.Errorf(
			"convert persisted store ID: %w",
			err,
		)
	}

	ownerID, err := fromUUID(row.OwnerID)
	if err != nil {
		return nil, fmt.Errorf(
			"convert persisted owner ID: %w",
			err,
		)
	}

	var imageReference *string

	if row.ImageReference.Valid {
		value := row.ImageReference.String
		imageReference = &value
	}

	store, err := entities.ReconstituteStore(
		storeID,
		ownerID,
		row.Name,
		row.Slug,
		row.Description,
		imageReference,
		row.Plan,
		row.Status,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstitute store aggregate: %w",
			err,
		)
	}

	return store, nil
}

// toUUID converts a standard UUID into pgx's PostgreSQL UUID type.
func toUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: value,
		Valid: true,
	}
}

// fromUUID converts pgx's PostgreSQL UUID type into a standard UUID.
func fromUUID(value pgtype.UUID) (uuid.UUID, error) {
	if !value.Valid {
		return uuid.Nil, errors.New(
			"persisted UUID is NULL",
		)
	}

	return uuid.UUID(value.Bytes), nil
}

// toTimestamp converts a Go timestamp into pgx's PostgreSQL timestamp type.
func toTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value.UTC(),
		Valid: true,
	}
}

// toNullableText converts an optional image reference into PostgreSQL NULL
// or TEXT.
func toNullableText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: *value,
		Valid:  true,
	}
}

// translateReadError converts PostgreSQL lookup errors into application
// semantics while preserving unexpected infrastructure failures.
func translateReadError(
	operation string,
	err error,
) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf(
			"%w: %s",
			applicationerrors.ErrStoreNotFound,
			operation,
		)
	}

	return fmt.Errorf(
		"%s: %w",
		operation,
		err,
	)
}

// translateWriteError converts known PostgreSQL unique violations into the
// semantic application errors expected by the Store application layer.
func translateWriteError(
	operation string,
	err error,
) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "stores_slug_key":
			return fmt.Errorf(
				"%w: %s",
				applicationerrors.ErrStoreSlugAlreadyExists,
				operation,
			)

		case "stores_owner_id_unique":
			return fmt.Errorf(
				"%w: %s",
				applicationerrors.ErrStoreAlreadyExists,
				operation,
			)
		}
	}

	return fmt.Errorf(
		"%s: %w",
		operation,
		err,
	)
}
