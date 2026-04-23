-- +goose Up
-- One rating per dinner. If a dinner has been rated multiple times (the old
-- record_retrospective tool appended rows), collapse to the most recent.
DELETE FROM dish_ratings a
USING dish_ratings b
WHERE a.week_dinner_id = b.week_dinner_id
  AND a.created_at < b.created_at;

CREATE UNIQUE INDEX dish_ratings_week_dinner_id_key ON dish_ratings(week_dinner_id);

-- +goose Down
DROP INDEX IF EXISTS dish_ratings_week_dinner_id_key;
