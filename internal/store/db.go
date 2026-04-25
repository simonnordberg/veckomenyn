package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // register pgx sql driver for goose
	"github.com/pressly/goose/v3"

	"github.com/simonnordberg/veckomenyn/internal/migrations"
)

type DB struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

// Migrate applies any pending migrations from the embedded filesystem.
func Migrate(ctx context.Context, dsn string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql for migrate: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		return fmt.Errorf("up: %w", err)
	}
	return nil
}

// PendingMigrations reports the number of migrations not yet applied to the
// target DB and the current applied version. Returns (0, 0, nil) for fresh
// installs (goose's tracker table doesn't exist yet) so callers can skip
// pre-migration backups on first boot.
func PendingMigrations(ctx context.Context, dsn string) (count int, currentVersion int64, err error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return 0, 0, fmt.Errorf("open sql: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	// information_schema is universally readable and avoids any goose
	// state mutation just to detect "is there anything to migrate".
	var hasTracker bool
	row := sqlDB.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'goose_db_version'
		)
	`)
	if err := row.Scan(&hasTracker); err != nil {
		return 0, 0, fmt.Errorf("check tracker: %w", err)
	}
	if !hasTracker {
		return 0, 0, nil
	}

	if err := sqlDB.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version`).
		Scan(&currentVersion); err != nil {
		return 0, 0, fmt.Errorf("read current version: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, 0, fmt.Errorf("set dialect: %w", err)
	}
	migs, err := goose.CollectMigrations(".", 0, math.MaxInt64)
	if err != nil {
		return 0, 0, fmt.Errorf("collect migrations: %w", err)
	}
	for _, m := range migs {
		if m.Version > currentVersion {
			count++
		}
	}
	return count, currentVersion, nil
}
