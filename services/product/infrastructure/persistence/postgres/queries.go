package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	generated "github.com/iheanyi-dev/ecommerce-backend/services/product/infrastructure/persistence/postgres/generated"
)

// NewQueries constructs SQLC queries backed by the Product connection pool.
func NewQueries(pool *pgxpool.Pool) *generated.Queries {
	return generated.New(pool)
}
