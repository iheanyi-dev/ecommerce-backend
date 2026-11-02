package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/iheanyi-dev/ecommerce-backend/services/payment/infrastructure/persistence/postgres"
	"github.com/iheanyi-dev/ecommerce-backend/services/payment/shared/config"
	"go.uber.org/fx"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func validDatabaseConfig() *config.Config {
	return &config.Config{
		DatabaseHost:     "localhost",
		DatabasePort:     "5432",
		DatabaseUser:     "payment",
		DatabasePassword: "test-password",
		DatabaseName:     "payment",
		DatabaseSSLMode:  "disable",
	}
}

func TestNewPoolRejectsNilConfig(t *testing.T) {
	pool, err := postgres.NewPool(&testLifecycle{}, nil)

	if err == nil {
		t.Fatal("expected an error for nil config")
	}

	if pool != nil {
		t.Fatal("expected nil pool")
	}
}

func TestNewPoolRejectsNilLifecycle(t *testing.T) {
	pool, err := postgres.NewPool(nil, validDatabaseConfig())

	if err == nil {
		t.Fatal("expected an error for nil lifecycle")
	}

	if pool != nil {
		t.Fatal("expected nil pool")
	}
}

func TestNewPoolRejectsInvalidDatabaseConfig(t *testing.T) {
	cfg := validDatabaseConfig()
	cfg.DatabasePort = "invalid-port"

	pool, err := postgres.NewPool(&testLifecycle{}, cfg)

	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected an error for invalid database configuration")
	}

	if pool != nil {
		pool.Close()
		t.Fatal("expected nil pool")
	}
}

func TestNewPoolConfiguresConnectionLimits(t *testing.T) {
	lifecycle := &testLifecycle{}

	pool, err := postgres.NewPool(lifecycle, validDatabaseConfig())
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	poolConfig := pool.Config()

	if poolConfig.MaxConns != 10 {
		t.Errorf("expected MaxConns 10, got %d", poolConfig.MaxConns)
	}

	if poolConfig.MinConns != 2 {
		t.Errorf("expected MinConns 2, got %d", poolConfig.MinConns)
	}

	if poolConfig.MaxConnLifetime != time.Hour {
		t.Errorf("unexpected MaxConnLifetime: %v", poolConfig.MaxConnLifetime)
	}

	if poolConfig.MaxConnIdleTime != 30*time.Minute {
		t.Errorf("unexpected MaxConnIdleTime: %v", poolConfig.MaxConnIdleTime)
	}

	if poolConfig.HealthCheckPeriod != time.Minute {
		t.Errorf("unexpected HealthCheckPeriod: %v", poolConfig.HealthCheckPeriod)
	}
}

func TestNewPoolRegistersLifecycleHooks(t *testing.T) {
	lifecycle := &testLifecycle{}

	pool, err := postgres.NewPool(lifecycle, validDatabaseConfig())
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	if len(lifecycle.hooks) != 1 {
		pool.Close()
		t.Fatalf("expected 1 lifecycle hook, got %d", len(lifecycle.hooks))
	}

	if lifecycle.hooks[0].OnStart == nil {
		pool.Close()
		t.Fatal("expected startup hook")
	}

	if lifecycle.hooks[0].OnStop == nil {
		pool.Close()
		t.Fatal("expected shutdown hook")
	}

	if err := lifecycle.hooks[0].OnStop(context.Background()); err != nil {
		t.Fatalf("shutdown pool: %v", err)
	}
}
