package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdateConfig is the singleton update_config row.
type UpdateConfig struct {
	AutoUpdateEnabled bool
}

// UpdateConfigStore reads and writes the singleton row driving in-app
// update behavior (manual button always works; auto-update toggle gates
// the daily scheduler).
type UpdateConfigStore struct {
	pool *pgxpool.Pool
}

func NewUpdateConfigStore(pool *pgxpool.Pool) *UpdateConfigStore {
	return &UpdateConfigStore{pool: pool}
}

func (u *UpdateConfigStore) UpdateConfig(ctx context.Context) (UpdateConfig, error) {
	var c UpdateConfig
	err := u.pool.QueryRow(ctx,
		`SELECT auto_update_enabled FROM update_config WHERE id = TRUE`).
		Scan(&c.AutoUpdateEnabled)
	if err != nil {
		return UpdateConfig{}, fmt.Errorf("read update_config: %w", err)
	}
	return c, nil
}

// AutoUpdateEnabled implements updates.ConfigReader so the auto-update
// scheduler can poll without owning the larger UpdateConfig type.
func (u *UpdateConfigStore) AutoUpdateEnabled(ctx context.Context) (bool, error) {
	c, err := u.UpdateConfig(ctx)
	if err != nil {
		return false, err
	}
	return c.AutoUpdateEnabled, nil
}

func (u *UpdateConfigStore) SetAutoUpdate(ctx context.Context, enabled bool) (UpdateConfig, error) {
	if _, err := u.pool.Exec(ctx, `
		UPDATE update_config
		SET auto_update_enabled = $1, updated_at = now()
		WHERE id = TRUE
	`, enabled); err != nil {
		return UpdateConfig{}, fmt.Errorf("update auto_update_enabled: %w", err)
	}
	return u.UpdateConfig(ctx)
}
