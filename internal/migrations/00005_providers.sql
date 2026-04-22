-- +goose Up

-- Polymorphic provider table. Each row is one integration (LLM provider,
-- shopping provider, etc.). Kind-specific fields live in config_json so
-- adding a new provider (e.g. ICA, OpenAI) is a new row, not a schema change.
--
-- Secrets stored in cleartext for now; self-hosted family deployment where
-- DB access == app access. A future migration can wrap values with AES-GCM
-- keyed by a MASTER_KEY env var without changing the row shape.
--
-- Convention: secret keys in config_json follow naming (api_key, password,
-- token, secret) so the REST layer can mask them on read.

CREATE TABLE providers (
    kind TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT false,
    config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER tg_providers_updated
    BEFORE UPDATE ON providers
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose Down

DROP TABLE IF EXISTS providers;
