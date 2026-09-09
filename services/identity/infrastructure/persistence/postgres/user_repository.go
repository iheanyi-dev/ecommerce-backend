package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	application_errors "github.com/iheanyi-dev/ecommerce-backend/services/identity/application/errors"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserRepository is the PostgreSQL implementation of the Identity
// service's UserRepository application port.
//
// Infrastructure is responsible for translating between the domain
// aggregate and PostgreSQL's persistence representation.
//
// It is also the boundary at which PostgreSQL-specific errors are
// translated into semantic application errors.
type UserRepository struct {
	queries *generated.Queries
}

// NewUserRepository creates a PostgreSQL UserRepository using the
// SQLC-generated query executor.
func NewUserRepository(
	queries *generated.Queries,
) *UserRepository {
	return &UserRepository{
		queries: queries,
	}
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

func toPgTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}

// Create persists a newly created User aggregate.
//
// The User aggregate has already been validated by the domain layer.
// This method therefore focuses on translating the aggregate into the
// parameters required by the SQLC-generated query.
//
// PostgreSQL remains the final authority for uniqueness. The repository
// translates the users_email_key violation into the semantic application
// error ErrEmailAlreadyExists.
//
// This is important because ExistsByEmail() is only a pre-check and cannot
// eliminate a race condition between two concurrent registration requests.
func (r *UserRepository) Create(
	ctx context.Context,
	newUser *user.User,
) error {
	params := generated.CreateUserParams{
		ID:           toPgUUID(newUser.ID().Value()),
		FullName:     newUser.FullName().String(),
		Email:        newUser.Email().String(),
		PasswordHash: newUser.PasswordHash().String(),
		Role:         newUser.Role().String(),
		Status:       newUser.Status().String(),
		CreatedAt:    toPgTimestamp(newUser.CreatedAt()),
		UpdatedAt:    toPgTimestamp(newUser.UpdatedAt()),
	}

	err := r.queries.CreateUser(ctx, params)
	if err == nil {
		return nil
	}

	// PostgreSQL reports unique-constraint violations using SQLSTATE
	// 23505. We additionally verify the constraint name so that another
	// future unique constraint is not incorrectly interpreted as a
	// duplicate email.
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "users_email_key" {
		return fmt.Errorf(
			"create user: %w",
			application_errors.ErrEmailAlreadyExists,
		)
	}

	// Unexpected persistence failures remain wrapped infrastructure
	// errors. The underlying error is preserved for diagnostics without
	// leaking PostgreSQL-specific knowledge into higher layers.
	return fmt.Errorf(
		"create user: %w",
		err,
	)
}

// ExistsByEmail checks whether a user with the supplied email already
// exists in the Identity database.
//
// Only the boolean result is required by registration, so we deliberately
// avoid loading an entire User aggregate.
func (r *UserRepository) ExistsByEmail(
	ctx context.Context,
	email user.Email,
) (bool, error) {
	count, err := r.queries.ExistsUserByEmail(
		ctx,
		email.String(),
	)
	if err != nil {
		return false, fmt.Errorf(
			"check user email existence: %w",
			err,
		)
	}

	return count, nil
}

// FindByEmail retrieves a persisted User aggregate by email.
//
// SQLC returns primitive persistence values. This method is responsible
// for translating those values back into domain value objects before
// reconstructing the User aggregate.
//
// PostgreSQL's no-row condition is translated into the application-level
// ErrUserNotFound so the application layer never depends on pgx.
func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	row, err := r.queries.FindUserByEmail(
		ctx,
		email.String(),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"find user by email: %w",
				application_errors.ErrUserNotFound,
			)
		}

		return nil, fmt.Errorf(
			"find user by email: %w",
			err,
		)
	}

	id, err := user.UserIDFromString(row.ID.String())
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user ID: %w",
			err,
		)
	}

	fullName, err := user.NewFullName(row.FullName)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct full name: %w",
			err,
		)
	}

	persistedEmail, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct email: %w",
			err,
		)
	}

	passwordHash, err := user.NewPasswordHash(row.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct password hash: %w",
			err,
		)
	}

	role, err := user.NewRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user role: %w",
			err,
		)
	}

	status, err := user.NewStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user status: %w",
			err,
		)
	}

	reconstitutedUser := user.ReconstituteUser(
		id,
		fullName,
		persistedEmail,
		passwordHash,
		role,
		status,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)

	return reconstitutedUser, nil
}

// FindByID retrieves a persisted User aggregate by UserID.
//
// SQLC returns primitive persistence values. This method translates those
// values back to domain value objects before reconstructing the domain
// aggregate.
//
// PostgreSQL's no-row condition is translated into the application-level
// ErrUserNotFound so the application layer never depends on pgx.
func (r *UserRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	row, err := r.queries.FindUserByID(
		ctx,
		toPgUUID(id.Value()),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"find user by ID: %w",
				application_errors.ErrUserNotFound,
			)
		}

		return nil, fmt.Errorf(
			"find user by ID: %w",
			err,
		)
	}

	userID, err := user.UserIDFromString(row.ID.String())
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user ID: %w",
			err,
		)
	}

	fullName, err := user.NewFullName(row.FullName)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct full name: %w",
			err,
		)
	}

	email, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct email: %w",
			err,
		)
	}

	passwordHash, err := user.NewPasswordHash(row.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct password hash: %w",
			err,
		)
	}

	role, err := user.NewRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user role: %w",
			err,
		)
	}

	status, err := user.NewStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user status: %w",
			err,
		)
	}

	reconstitutedUser := user.ReconstituteUser(
		userID,
		fullName,
		email,
		passwordHash,
		role,
		status,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)

	return reconstitutedUser, nil
}

// List retrieves users from PostgreSQL using limit/offset pagination.
//
// SQLC returns persistence models, so this method is responsible for
// reconstructing each record into the corresponding domain User aggregate.
// The application layer therefore remains completely independent of SQLC.
func (r *UserRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]*user.User, error) {
	rows, err := r.queries.ListUsers(
		ctx,
		generated.ListUsersParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list users: %w",
			err,
		)
	}

	users := make([]*user.User, 0, len(rows))

	for _, row := range rows {
		userID, err := user.UserIDFromString(row.ID.String())
		if err != nil {
			return nil, fmt.Errorf(
				"reconstruct user ID: %w",
				err,
			)
		}

		fullName, err := user.NewFullName(row.FullName)
		if err != nil {
			return nil, fmt.Errorf(
				"reconstruct full name: %w",
				err,
			)
		}

		email, err := user.NewEmail(row.Email)
		if err != nil {
			return nil, fmt.Errorf(
				"reconstruct email: %w",
				err,
			)
		}

		passwordHash, err := user.NewPasswordHash(row.PasswordHash)
		if err != nil {
			return nil, fmt.Errorf(
				"reconstruct password hash: %w",
				err,
			)
		}

		role, err := user.NewRole(row.Role)
		if err != nil {
			return nil, fmt.Errorf(
				"reconstruct user role: %w",
				err,
			)
		}

		status, err := user.NewStatus(row.Status)
		if err != nil {
			return nil, fmt.Errorf(
				"reconstruct user status: %w",
				err,
			)
		}

		reconstitutedUser := user.ReconstituteUser(
			userID,
			fullName,
			email,
			passwordHash,
			role,
			status,
			row.CreatedAt.Time,
			row.UpdatedAt.Time,
		)

		users = append(users, reconstitutedUser)
	}

	return users, nil
}

// UpdateFullName persists the changed full name and the UpdatedAt timestamp
// generated by the domain aggregate.
//
// The repository does not generate the timestamp. The User aggregate owns
// that state, and persistence simply stores it.
func (r *UserRepository) UpdateFullName(
	ctx context.Context,
	existingUser *user.User,
) error {
	err := r.queries.UpdateUserFullName(
		ctx,
		generated.UpdateUserFullNameParams{
			ID:        toPgUUID(existingUser.ID().Value()),
			FullName:  existingUser.FullName().String(),
			UpdatedAt: toPgTimestamp(existingUser.UpdatedAt()),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"update user full name: %w",
			err,
		)
	}

	return nil
}

// UpdatePasswordHash persists the changed password hash and the UpdatedAt
// timestamp generated by the domain aggregate.
//
// Plaintext passwords never reach this repository. The application layer
// hashes the password first, then the domain aggregate stores the resulting
// PasswordHash.
func (r *UserRepository) UpdatePasswordHash(
	ctx context.Context,
	existingUser *user.User,
) error {
	err := r.queries.UpdateUserPasswordHash(
		ctx,
		generated.UpdateUserPasswordHashParams{
			ID:           toPgUUID(existingUser.ID().Value()),
			PasswordHash: existingUser.PasswordHash().String(),
			UpdatedAt:    toPgTimestamp(existingUser.UpdatedAt()),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"update user password hash: %w",
			err,
		)
	}

	return nil
}

// UpdateStatus persists the changed account status and the UpdatedAt
// timestamp generated by the domain aggregate.
//
// The repository does not generate the timestamp. The User aggregate owns
// that state, and persistence simply stores it.
func (r *UserRepository) UpdateStatus(
	ctx context.Context,
	existingUser *user.User,
) error {
	err := r.queries.UpdateUserStatus(
		ctx,
		generated.UpdateUserStatusParams{
			ID:        toPgUUID(existingUser.ID().Value()),
			Status:    string(existingUser.Status()),
			UpdatedAt: toPgTimestamp(existingUser.UpdatedAt()),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"update user status: %w",
			err,
		)
	}

	return nil
}

// Compile-time assertion.
//
// This guarantees that the infrastructure implementation always satisfies
// the application contract.
var _ ports.UserRepository = (*UserRepository)(nil)
