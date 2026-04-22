-- +goose Up

-- Household-wide defaults that shape how new weeks get planned.
-- Singleton row (id = 1). Held in a dedicated table rather than key/value
-- so the columns are typed and the PlanNewForm can bind directly.

CREATE TABLE household_settings (
    id INT PRIMARY KEY CHECK (id = 1),
    default_dinners INT NOT NULL DEFAULT 7 CHECK (default_dinners BETWEEN 1 AND 14),
    -- ISO weekday: 1 = Monday … 7 = Sunday.
    default_delivery_weekday INT NOT NULL DEFAULT 1 CHECK (default_delivery_weekday BETWEEN 1 AND 7),
    -- Days between order and delivery. -1 means order the day before delivery (typical).
    default_order_offset_days INT NOT NULL DEFAULT -1,
    default_servings INT NOT NULL DEFAULT 4 CHECK (default_servings BETWEEN 1 AND 20),
    notes_md TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER tg_household_settings_updated
    BEFORE UPDATE ON household_settings
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

INSERT INTO household_settings (id) VALUES (1);

-- +goose Down

DROP TABLE IF EXISTS household_settings;
