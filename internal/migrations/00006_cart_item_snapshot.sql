-- +goose Up
-- Snapshot of product details at the time a line was added, so the UI and
-- future retrospectives can render names and prices without a live lookup.
-- Shape: {"name": string, "unit_price": number, "line_total": number,
--         "unit": string}. Empty object means "imported from history" or
--         "added before the snapshot column existed".
ALTER TABLE cart_items
    ADD COLUMN product_snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE cart_items DROP COLUMN product_snapshot_json;
