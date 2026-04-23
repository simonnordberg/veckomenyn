-- +goose Up
-- Collapse 'archived' into 'ordered'. Manual lifecycle; no auto-archive.
-- Ratings and retrospectives gate on 'ordered' going forward.
UPDATE weeks SET status = 'ordered' WHERE status = 'archived';

ALTER TABLE weeks DROP CONSTRAINT weeks_status_check;
ALTER TABLE weeks ADD CONSTRAINT weeks_status_check
    CHECK (status IN ('draft', 'cart_built', 'ordered'));

-- +goose Down
ALTER TABLE weeks DROP CONSTRAINT weeks_status_check;
ALTER TABLE weeks ADD CONSTRAINT weeks_status_check
    CHECK (status IN ('draft', 'cart_built', 'ordered', 'archived'));
