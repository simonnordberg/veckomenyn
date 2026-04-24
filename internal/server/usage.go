package server

import (
	"net/http"
	"time"
)

// usageSummary collects everything the admin usage page renders in one call.
// All figures cover a rolling 30-day window so a single endpoint hit gives
// the full picture; older history is still in the table for ad-hoc SQL.
type usageSummary struct {
	WindowDays int             `json:"window_days"`
	Total      usageTotals     `json:"total"`
	ByDay      []dailyUsage    `json:"by_day"`
	ByModel    []modelUsage    `json:"by_model"`
	ByWeek     []weekUsage     `json:"by_week"`
	RecentConv []conversationUsage `json:"recent_conversations"`
}

type usageTotals struct {
	CostUSD                  float64 `json:"cost_usd"`
	InputTokens              int64   `json:"input_tokens"`
	CacheCreationInputTokens int64   `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64   `json:"cache_read_input_tokens"`
	OutputTokens             int64   `json:"output_tokens"`
	Calls                    int64   `json:"calls"`
}

type dailyUsage struct {
	Date    string  `json:"date"`
	CostUSD float64 `json:"cost_usd"`
	Calls   int64   `json:"calls"`
}

type modelUsage struct {
	Model                    string  `json:"model"`
	CostUSD                  float64 `json:"cost_usd"`
	InputTokens              int64   `json:"input_tokens"`
	CacheCreationInputTokens int64   `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64   `json:"cache_read_input_tokens"`
	OutputTokens             int64   `json:"output_tokens"`
	Calls                    int64   `json:"calls"`
}

type weekUsage struct {
	WeekID  int64   `json:"week_id"`
	IsoWeek string  `json:"iso_week"`
	CostUSD float64 `json:"cost_usd"`
	Calls   int64   `json:"calls"`
}

type conversationUsage struct {
	ConversationID int64   `json:"conversation_id"`
	Title          string  `json:"title"`
	WeekID         *int64  `json:"week_id,omitempty"`
	IsoWeek        string  `json:"iso_week,omitempty"`
	CostUSD        float64 `json:"cost_usd"`
	Calls          int64   `json:"calls"`
	LastUsedAt     string  `json:"last_used_at"`
}

func (s *Server) handleGetUsageSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	const windowDays = 30
	since := time.Now().AddDate(0, 0, -windowDays).Format(time.RFC3339)

	var out usageSummary
	out.WindowDays = windowDays

	// Totals across the window.
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0)::float8,
		       COALESCE(SUM(input_tokens), 0),
		       COALESCE(SUM(cache_creation_input_tokens), 0),
		       COALESCE(SUM(cache_read_input_tokens), 0),
		       COALESCE(SUM(output_tokens), 0),
		       COUNT(*)
		FROM llm_usage
		WHERE created_at >= $1`, since).Scan(
		&out.Total.CostUSD, &out.Total.InputTokens,
		&out.Total.CacheCreationInputTokens, &out.Total.CacheReadInputTokens,
		&out.Total.OutputTokens, &out.Total.Calls,
	)
	if err != nil {
		s.internalError(w, r, "usage totals", err)
		return
	}

	// By day.
	rows, err := s.db.Pool.Query(ctx, `
		SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
		       COALESCE(SUM(cost_usd), 0)::float8,
		       COUNT(*)
		FROM llm_usage
		WHERE created_at >= $1
		GROUP BY day
		ORDER BY day`, since)
	if err != nil {
		s.internalError(w, r, "usage by day", err)
		return
	}
	for rows.Next() {
		var d dailyUsage
		if err := rows.Scan(&d.Date, &d.CostUSD, &d.Calls); err != nil {
			rows.Close()
			s.internalError(w, r, "usage by day scan", err)
			return
		}
		out.ByDay = append(out.ByDay, d)
	}
	rows.Close()
	if out.ByDay == nil {
		out.ByDay = []dailyUsage{}
	}

	// By model.
	rows, err = s.db.Pool.Query(ctx, `
		SELECT model,
		       COALESCE(SUM(cost_usd), 0)::float8,
		       COALESCE(SUM(input_tokens), 0),
		       COALESCE(SUM(cache_creation_input_tokens), 0),
		       COALESCE(SUM(cache_read_input_tokens), 0),
		       COALESCE(SUM(output_tokens), 0),
		       COUNT(*)
		FROM llm_usage
		WHERE created_at >= $1
		GROUP BY model
		ORDER BY SUM(cost_usd) DESC`, since)
	if err != nil {
		s.internalError(w, r, "usage by model", err)
		return
	}
	for rows.Next() {
		var m modelUsage
		if err := rows.Scan(&m.Model, &m.CostUSD, &m.InputTokens,
			&m.CacheCreationInputTokens, &m.CacheReadInputTokens,
			&m.OutputTokens, &m.Calls); err != nil {
			rows.Close()
			s.internalError(w, r, "usage by model scan", err)
			return
		}
		out.ByModel = append(out.ByModel, m)
	}
	rows.Close()
	if out.ByModel == nil {
		out.ByModel = []modelUsage{}
	}

	// By week (only rows where week_id was known at write time).
	rows, err = s.db.Pool.Query(ctx, `
		SELECT u.week_id,
		       COALESCE(w.iso_week, '') AS iso_week,
		       COALESCE(SUM(u.cost_usd), 0)::float8,
		       COUNT(*)
		FROM llm_usage u
		LEFT JOIN weeks w ON w.id = u.week_id
		WHERE u.created_at >= $1 AND u.week_id IS NOT NULL
		GROUP BY u.week_id, w.iso_week
		ORDER BY SUM(u.cost_usd) DESC
		LIMIT 20`, since)
	if err != nil {
		s.internalError(w, r, "usage by week", err)
		return
	}
	for rows.Next() {
		var v weekUsage
		if err := rows.Scan(&v.WeekID, &v.IsoWeek, &v.CostUSD, &v.Calls); err != nil {
			rows.Close()
			s.internalError(w, r, "usage by week scan", err)
			return
		}
		out.ByWeek = append(out.ByWeek, v)
	}
	rows.Close()
	if out.ByWeek == nil {
		out.ByWeek = []weekUsage{}
	}

	// Most expensive recent conversations (top 20 by cost in window).
	rows, err = s.db.Pool.Query(ctx, `
		SELECT u.conversation_id,
		       COALESCE(c.title, ''),
		       u.week_id,
		       COALESCE(w.iso_week, ''),
		       COALESCE(SUM(u.cost_usd), 0)::float8,
		       COUNT(*),
		       MAX(u.created_at)::text
		FROM llm_usage u
		LEFT JOIN conversations c ON c.id = u.conversation_id
		LEFT JOIN weeks w ON w.id = u.week_id
		WHERE u.created_at >= $1 AND u.conversation_id IS NOT NULL
		GROUP BY u.conversation_id, c.title, u.week_id, w.iso_week
		ORDER BY SUM(u.cost_usd) DESC
		LIMIT 20`, since)
	if err != nil {
		s.internalError(w, r, "usage by conversation", err)
		return
	}
	for rows.Next() {
		var c conversationUsage
		var weekID *int64
		if err := rows.Scan(&c.ConversationID, &c.Title, &weekID, &c.IsoWeek, &c.CostUSD, &c.Calls, &c.LastUsedAt); err != nil {
			rows.Close()
			s.internalError(w, r, "usage by conversation scan", err)
			return
		}
		c.WeekID = weekID
		out.RecentConv = append(out.RecentConv, c)
	}
	rows.Close()
	if out.RecentConv == nil {
		out.RecentConv = []conversationUsage{}
	}

	writeJSON(w, http.StatusOK, out)
}
