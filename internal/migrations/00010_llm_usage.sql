-- +goose Up

-- One row per model call (i.e. per tool-use iteration inside Agent.Run).
-- conversation_id lets us roll up per chat session; week_id is denormalized
-- at write time so weekly rollups don't need a join and survive the
-- conversation being detached from the week later.
CREATE TABLE llm_usage (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE SET NULL,
    week_id BIGINT REFERENCES weeks(id) ON DELETE SET NULL,
    model TEXT NOT NULL,
    input_tokens INT NOT NULL,
    cache_creation_input_tokens INT NOT NULL,
    cache_read_input_tokens INT NOT NULL,
    output_tokens INT NOT NULL,
    cost_usd NUMERIC(12,6) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_llm_usage_conv ON llm_usage(conversation_id, created_at);
CREATE INDEX idx_llm_usage_week ON llm_usage(week_id, created_at);
CREATE INDEX idx_llm_usage_created ON llm_usage(created_at);

-- +goose Down

DROP TABLE IF EXISTS llm_usage;
