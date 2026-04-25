package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/backup"
)

// BackupConfigStore reads and writes the singleton backup_config row.
// Implements backup.ConfigReader so the scheduler can poll it.
type BackupConfigStore struct {
	pool *pgxpool.Pool
}

func NewBackupConfigStore(pool *pgxpool.Pool) *BackupConfigStore {
	return &BackupConfigStore{pool: pool}
}

func (b *BackupConfigStore) BackupConfig(ctx context.Context) (backup.Config, error) {
	var cfg backup.Config
	err := b.pool.QueryRow(ctx,
		`SELECT nightly_enabled, nightly_keep FROM backup_config WHERE id = TRUE`).
		Scan(&cfg.NightlyEnabled, &cfg.NightlyKeep)
	if err != nil {
		return backup.Config{}, fmt.Errorf("read backup_config: %w", err)
	}
	return cfg, nil
}

// UpdateBackupConfig partially updates the singleton row. Only fields whose
// pointer is non-nil are written.
func (b *BackupConfigStore) UpdateBackupConfig(ctx context.Context, nightlyEnabled *bool, nightlyKeep *int) (backup.Config, error) {
	if _, err := b.pool.Exec(ctx, `
		UPDATE backup_config
		SET nightly_enabled = COALESCE($1, nightly_enabled),
		    nightly_keep    = COALESCE($2, nightly_keep),
		    updated_at      = now()
		WHERE id = TRUE
	`, nightlyEnabled, nightlyKeep); err != nil {
		return backup.Config{}, fmt.Errorf("update backup_config: %w", err)
	}
	return b.BackupConfig(ctx)
}
