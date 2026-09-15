package gateway_test

import (
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/gateway"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestModule_ProvidesGatewayDependencies(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_PORT", "8081")
	t.Setenv("IDENTITY_SERVICE_URL", "http://localhost:8080")

	app := fx.New(
		gateway.Module,
		fx.NopLogger,
	)

	require.NoError(t, app.Err())
}
