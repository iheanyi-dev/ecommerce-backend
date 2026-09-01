package postgres_tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/shared/config"
	"github.com/iheanyi-dev/ecommerce-backend/services/identity/infrastructure/persistence/postgres"
)

func TestNewPool_ConnectsToPostgres(t *testing.T) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	cfg, err := config.Load()
	require.NoError(t, err)

	pool, err := postgres.NewPool(cfg)
	require.NoError(t, err)

	defer pool.Close()

	require.NoError(t, pool.Ping(ctx))
}