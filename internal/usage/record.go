package usage

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/llm"
)

// Recorder inserts one llm_usage row per model call. It captures the
// denormalized week_id at write time so weekly rollups survive the
// conversation being detached from a week later.
type Recorder struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewRecorder(db *pgxpool.Pool, log *slog.Logger) *Recorder {
	return &Recorder{db: db, log: log}
}

// Record writes a single usage row. convID and weekID may be zero for
// calls not tied to a specific conversation or week (in which case they're
// stored as SQL NULL via nullable helpers below).
//
// Failures are logged and swallowed: instrumentation must never break the
// user-facing chat flow.
func (r *Recorder) Record(ctx context.Context, convID, weekID int64, model string, u llm.Usage) {
	if r == nil || r.db == nil {
		return
	}
	if !KnownModel(model) {
		r.log.Warn("llm_usage: unknown model, cost will be zero", "model", model)
	}
	cost := Cost(model, u)

	_, err := r.db.Exec(ctx, `
		INSERT INTO llm_usage (
			conversation_id, week_id, model,
			input_tokens, cache_creation_input_tokens, cache_read_input_tokens, output_tokens,
			cost_usd
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		nullInt64(convID), nullInt64(weekID), model,
		u.InputTokens, u.CacheCreationInputTokens, u.CacheReadInputTokens, u.OutputTokens,
		cost,
	)
	if err != nil {
		r.log.Error("llm_usage: insert failed", "err", err, "model", model, "conv", convID, "week", weekID)
	}
}

func nullInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}
