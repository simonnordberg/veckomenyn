package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidDinnerRatings is the closed set accepted by dish_ratings.rating.
// Kept in sync with the CHECK constraint in migration 00001.
var ValidDinnerRatings = map[string]bool{
	"loved":    true,
	"liked":    true,
	"meh":      true,
	"disliked": true,
}

// SetDinnerRating upserts a rating for a dinner. One row per week_dinner_id.
// Returns ErrNoRows if the dinner doesn't exist.
func SetDinnerRating(ctx context.Context, pool *pgxpool.Pool, dinnerID int64, rating, notes string) error {
	if !ValidDinnerRatings[rating] {
		return fmt.Errorf("invalid rating %q", rating)
	}
	tag, err := pool.Exec(ctx, `
		INSERT INTO dish_ratings (week_dinner_id, rating, notes)
		SELECT id, $2, $3 FROM week_dinners WHERE id = $1
		ON CONFLICT (week_dinner_id) DO UPDATE
		SET rating = EXCLUDED.rating, notes = EXCLUDED.notes`,
		dinnerID, rating, notes)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("dinner %d not found", dinnerID)
	}
	return nil
}

// ClearDinnerRating removes the rating for a dinner, if any. No-op if none.
func ClearDinnerRating(ctx context.Context, pool *pgxpool.Pool, dinnerID int64) error {
	_, err := pool.Exec(ctx, `DELETE FROM dish_ratings WHERE week_dinner_id = $1`, dinnerID)
	return err
}
