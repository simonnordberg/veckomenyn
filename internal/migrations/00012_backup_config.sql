-- +goose Up

-- backup_config is a singleton row driving the in-app scheduled backup.
-- Replaces the prodrigestivill sidecar with something users can manage
-- without editing compose.
CREATE TABLE backup_config (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id = TRUE), -- enforce singleton
    nightly_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    nightly_keep INT NOT NULL DEFAULT 14 CHECK (nightly_keep > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO backup_config (id) VALUES (TRUE);

-- +goose Down
DROP TABLE backup_config;
