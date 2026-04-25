// Package seed embeds starter preference markdown so the first-run wizard
// can populate the cooking_principles table without shelling out to the
// veckomenyn-import CLI.
package seed

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed preferences/*.md
var preferencesFS embed.FS

// Preferences upserts each shipped preference markdown file into
// cooking_principles, keyed by the filename stem. Returns the number of
// rows touched. Idempotent — re-seeding overwrites existing rows.
func Preferences(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	entries, err := fs.ReadDir(preferencesFS, "preferences")
	if err != nil {
		return 0, fmt.Errorf("read embed: %w", err)
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		body, err := fs.ReadFile(preferencesFS, "preferences/"+e.Name())
		if err != nil {
			return count, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		category := strings.TrimSuffix(filepath.Base(e.Name()), ".md")

		tx, err := pool.Begin(ctx)
		if err != nil {
			return count, fmt.Errorf("begin: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM cooking_principles WHERE category = $1`, category); err != nil {
			_ = tx.Rollback(ctx)
			return count, fmt.Errorf("delete %s: %w", category, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO cooking_principles (category, body_md) VALUES ($1, $2)`,
			category, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return count, fmt.Errorf("insert %s: %w", category, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return count, fmt.Errorf("commit %s: %w", category, err)
		}
		count++
	}
	return count, nil
}
