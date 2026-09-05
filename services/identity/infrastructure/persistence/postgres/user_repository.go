package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/ports"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/domain/user"
	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

// UserRepository is the PostgreSQL implementation of the Identity
// service's UserRepository application port.
//
// Infrastructure is responsible for translating between the domain
// aggregate and PostgreSQL's persistence representation.
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

	return r.queries.CreateUser(ctx, params)
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
		ctx, email.String(),
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
// The reconstruction flow is:
//
// PostgreSQL
//
//	↓
//
// SQLC generated User
//
//	↓
//
// Domain value-object construction
//
//	↓
//
// user.ReconstituteUser()
//
//	↓
//
// Valid User aggregate
func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email user.Email,
) (*user.User, error) {
	// Retrieve the raw persisted user through the SQLC-generated query.
	row, err := r.queries.FindUserByEmail(
		ctx,
		email.String(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find user by email: %w",
			err,
		)
	}

	// Reconstruct the UserID from the UUID returned by PostgreSQL.
	//
	// SQLC represents the UUID using pgtype.UUID. The domain uses its
	// own UserID value object, so the repository performs the translation
	// at this infrastructure boundary.
	id, err := user.UserIDFromString(row.ID.String())
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user ID: %w",
			err,
		)
	}

	// Reconstruct the FullName value object.
	fullName, err := user.NewFullName(row.FullName)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct full name: %w",
			err,
		)
	}

	// Reconstruct the Email value object.
	persistedEmail, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct email: %w",
			err,
		)
	}

	// Reconstruct the PasswordHash value object.
	//
	// We do not hash the password here. The database already contains
	// the securely hashed password. We are simply restoring the domain
	// representation of that persisted hash.
	passwordHash, err := user.NewPasswordHash(row.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct password hash: %w",
			err,
		)
	}

	// Reconstruct the user's Role from its persisted representation.
	role, err := user.NewRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user role: %w",
			err,
		)
	}

	// Reconstruct the user's account Status.
	status, err := user.NewStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf(
			"reconstruct user status: %w",
			err,
		)
	}

	// All persisted values have now been translated into validated
	// domain components.
	//
	// ReconstituteUser deliberately does not apply creation defaults.
	// It restores the exact state that was persisted in the database.
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
// values back into domain value objects before reconstructing the User
// aggregate.
//
// The repository performs this translation at the infrastructure boundary
// so the application layer never depends on PostgreSQL or SQLC types.
func (r *UserRepository) FindByID(
	ctx context.Context,
	id user.UserID,
) (*user.User, error) {
	row, err := r.queries.FindUserByID(
		ctx,
		toPgUUID(id.Value()),
	)
	if err != nil {
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

// Compile-time assertion.
//
// This guarantees that the infrastructure implementation always satisfies
// the application contract.
var _ ports.UserRepository = (*UserRepository)(nil)
