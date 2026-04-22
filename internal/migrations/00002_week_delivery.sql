-- +goose Up

-- A week is anchored on when the Willys order is delivered: the menu runs
-- from delivery_date + 1 for roughly 7 days. order_date is when the user
-- actually places the order on willys.se (usually the same day or the day
-- before delivery). Both are nullable so drafts without a firm date still fit.

ALTER TABLE weeks
    ADD COLUMN delivery_date DATE,
    ADD COLUMN order_date DATE;

COMMENT ON COLUMN weeks.delivery_date IS 'Day groceries arrive. Menu runs delivery_date + 1 for ~7 days.';
COMMENT ON COLUMN weeks.order_date IS 'Day the order is placed on willys.se.';

CREATE INDEX idx_weeks_delivery_date ON weeks(delivery_date DESC NULLS LAST);

-- +goose Down

DROP INDEX IF EXISTS idx_weeks_delivery_date;
ALTER TABLE weeks
    DROP COLUMN IF EXISTS order_date,
    DROP COLUMN IF EXISTS delivery_date;
