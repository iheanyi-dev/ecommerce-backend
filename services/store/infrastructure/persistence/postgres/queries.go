package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	generated "github.com/iheanyi-dev/ecommerce-backend/services/store/infrastructure/persistence/postgres/generated"
)

// NewQueries creates the SQLC query object used by Store repositories.
//
// SQLC generated code depends on the pgx DBTX abstraction. Supplying the
// connection pool here keeps that generated persistence detail inside the
// Infrastructure layer.
//
// Repositories depend on the generated Queries type rather than constructing
// SQL statements themselves.
func NewQueries(
	pool *pgxpool.Pool,
) *generated.Queries {
	return generated.New(pool)
}
