-- +goose Up

-- iso_week is a human label, not an identity. Multiple plans can share the
-- same label (e.g. two drafts for the same shopping period, or overlapping
-- short trips). Identity moves to weeks.id.
ALTER TABLE weeks DROP CONSTRAINT weeks_iso_week_key;
CREATE INDEX idx_weeks_iso_week ON weeks(iso_week);

-- +goose Down

DROP INDEX idx_weeks_iso_week;
ALTER TABLE weeks ADD CONSTRAINT weeks_iso_week_key UNIQUE (iso_week);
