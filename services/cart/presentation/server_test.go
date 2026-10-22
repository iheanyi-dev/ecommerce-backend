package presentation

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/cart/shared/config"
	"go.uber.org/fx"
)

func TestHTTPServerStartsAndStopsWithFx(t *testing.T) {
	app := fx.New(
		fx.Provide(
			func() *config.Config {
				return &config.Config{HTTPAddr: "127.0.0.1:0"}
			},
			func() http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			},
		),
		fx.Invoke(RegisterHTTPServerLifecycle),
		fx.StartTimeout(5*time.Second),
		fx.StopTimeout(5*time.Second),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("Fx app construction failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		t.Fatalf("HTTP server failed to start: %v", err)
	}

	if err := app.Stop(ctx); err != nil {
		t.Fatalf("HTTP server failed to stop gracefully: %v", err)
	}
}
