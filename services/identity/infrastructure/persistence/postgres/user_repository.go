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

// Compile-time assertion.
//
// This guarantees that the infrastructure implementation always satisfies
// the application contract.
var _ ports.UserRepository = (*UserRepository)(nil)
