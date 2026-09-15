package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	gatewayhttp "github.com/iheanyi-dev/ecommerce-backend/gateway/presentation/http"
	"github.com/iheanyi-dev/ecommerce-backend/gateway/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		AppEnv:             "test",
		AppPort:            "9090",
		IdentityServiceURL: "http://localhost:8080",
	}

	handler := http.NewServeMux()

	server := gatewayhttp.NewServer(cfg, handler)

	require.NotNil(t, server)
}

func TestServer_StartAndShutdown(t *testing.T) {
	cfg := &config.Config{
		AppEnv:             "test",
		AppPort:            "0",
		IdentityServiceURL: "http://localhost:8080",
	}

	handler := http.NewServeMux()

	server := gatewayhttp.NewServer(cfg, handler)

	started := make(chan error, 1)

	go func() {
		started <- server.Start()
	}()

	// Give ListenAndServe enough time to bind the ephemeral port.
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	err := server.Shutdown(ctx)

	require.NoError(t, err)

	select {
	case err := <-started:
		assert.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(time.Second):
		t.Fatal("server did not shut down")
	}
}
