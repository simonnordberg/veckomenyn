package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UpsertWeekRetrospective updates the most recent retrospective row for a week
// if one exists, or inserts a new row. The week-level retrospective is a single
// evolving note (pacing, portion totals, overall vibe) — older rows remain in
// the table as history but aren't surfaced in the editable UI.
// Returns an error wrapping "not found" if the week id doesn't exist.
func UpsertWeekRetrospective(ctx context.Context, pool *pgxpool.Pool, weekID int64, notesMD string) error {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM weeks WHERE id = $1)`, weekID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("week %d not found", weekID)
	}

	tag, err := pool.Exec(ctx, `
		UPDATE retrospectives SET notes_md = $2
		WHERE id = (SELECT id FROM retrospectives WHERE week_id = $1 ORDER BY created_at DESC LIMIT 1)`,
		weekID, notesMD)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO retrospectives (week_id, notes_md) VALUES ($1, $2)`, weekID, notesMD)
	return err
}
