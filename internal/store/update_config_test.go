package store

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/simonnordberg/veckomenyn/internal/migrations"
)

// setupDB matches the pattern in internal/agent/tools_db_test.go: connect
// to TEST_DATABASE_URL, run migrations, return a pool. Tests skip when
// the env var is unset.
func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := t.Context()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialect: %v", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestUpdateConfig_DefaultsAndToggle(t *testing.T) {
	pool := setupDB(t)
	ctx := context.Background()
	store := NewUpdateConfigStore(pool)

	// Migration seeds the singleton row with auto_update_enabled = false.
	cfg, err := store.UpdateConfig(ctx)
	if err != nil {
		t.Fatalf("read default: %v", err)
	}
	if cfg.AutoUpdateEnabled {
		t.Error("default should be false")
	}

	// Flip on.
	cfg, err = store.SetAutoUpdate(ctx, true)
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !cfg.AutoUpdateEnabled {
		t.Error("after enable should be true")
	}

	// Flip off.
	cfg, err = store.SetAutoUpdate(ctx, false)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if cfg.AutoUpdateEnabled {
		t.Error("after disable should be false")
	}

	// AutoUpdateEnabled adapter mirrors the field.
	on, err := store.AutoUpdateEnabled(ctx)
	if err != nil {
		t.Fatalf("AutoUpdateEnabled: %v", err)
	}
	if on {
		t.Error("expected false from AutoUpdateEnabled adapter")
	}
}
