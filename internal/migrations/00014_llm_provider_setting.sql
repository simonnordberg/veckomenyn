-- +goose Up
ALTER TABLE household_settings
    ADD COLUMN llm_provider TEXT NOT NULL DEFAULT 'anthropic';

-- +goose Down
ALTER TABLE household_settings DROP COLUMN llm_provider;
