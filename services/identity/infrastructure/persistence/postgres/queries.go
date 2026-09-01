package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	generated "github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres/generated"
)

// NewQueries creates the SQLC query object using the PostgreSQL pool.
//
// SQLC generates the actual query implementations. This function only
// wires those generated queries to our application's database connection.
func NewQueries(pool *pgxpool.Pool) *generated.Queries {
	return generated.New(pool)
}
