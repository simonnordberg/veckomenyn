-- +goose Up

-- system_secrets holds server-managed secrets that the binary needs to
-- bootstrap itself before user input is available. Currently just the
-- master encryption key (for provider credentials), but the table is
-- generic so future server-managed secrets can land here without a
-- dedicated schema change.
CREATE TABLE system_secrets (
    name TEXT PRIMARY KEY,
    value BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE system_secrets;
