-- +goose Up

ALTER TABLE household_settings
    ADD COLUMN language TEXT NOT NULL DEFAULT 'sv' CHECK (language IN ('sv', 'en'));

COMMENT ON COLUMN household_settings.language IS
    'UI language and the language the agent writes generated content in. Existing content is not retranslated when this changes.';

-- +goose Down

ALTER TABLE household_settings DROP COLUMN IF EXISTS language;
