-- +goose Up

-- update_config is a singleton row driving the in-app update flow.
-- auto_update_enabled flips the daily auto-trigger on/off; the manual
-- button uses the same trigger endpoint regardless.
CREATE TABLE update_config (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id = TRUE),
    auto_update_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO update_config (id) VALUES (TRUE);

-- +goose Down
DROP TABLE update_config;
