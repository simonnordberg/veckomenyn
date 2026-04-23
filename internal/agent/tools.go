package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/shopping"
	"github.com/simonnordberg/veckomenyn/internal/store"
)

// Tool is the interface every agent tool implements.
type Tool interface {
	Name() string
	Def() anthropic.ToolParam
	Call(ctx context.Context, input json.RawMessage) (string, error)
}

// simpleTool adapts a plain function into the Tool interface so each tool
// definition stays inline with its handler.
type simpleTool struct {
	name string
	def  anthropic.ToolParam
	fn   func(ctx context.Context, input json.RawMessage) (string, error)
}

func (t *simpleTool) Name() string                                              { return t.name }
func (t *simpleTool) Def() anthropic.ToolParam                                  { return t.def }
func (t *simpleTool) Call(c context.Context, i json.RawMessage) (string, error) { return t.fn(c, i) }

func newTool(name, desc string, props map[string]any, required []string,
	fn func(ctx context.Context, input json.RawMessage) (string, error)) Tool {
	schema := anthropic.ToolInputSchemaParam{Properties: props}
	if len(required) > 0 {
		schema.Required = required
	}
	return &simpleTool{
		name: name,
		def: anthropic.ToolParam{
			Name:        name,
			Description: anthropic.String(desc),
			InputSchema: schema,
		},
		fn: fn,
	}
}

// registerTools builds the full tool set. All DB-backed tools close over the
// pool; shopping tools close over the configured provider (nil → unavailable).
func registerTools(db *pgxpool.Pool, shop shopping.Provider, log *slog.Logger) []Tool {
	tools := []Tool{
		readPreferencesTool(db),
		updatePreferenceTool(db),
		readHouseholdSettingsTool(db),
		updateHouseholdSettingsTool(db),
		listDishesRecentTool(db),
		searchHistoryTool(db),
		listWeeksTool(db),
		getWeekTool(db),
		createWeekTool(db),
		updateWeekTool(db),
		addDinnerTool(db),
		updateDinnerTool(db),
		deleteDinnerTool(db),
		addExceptionTool(db),
		recordRetrospectiveTool(db),
	}
	if shop != nil {
		tools = append(tools,
			shopSearchTool(shop),
			shopCartGetTool(shop),
			shopCartAddTool(shop, db),
			shopCartAddManyTool(shop, db),
			shopCartRemoveTool(shop, db),
			shopCartClearTool(shop, db),
			shopOrdersRecentTool(shop),
			shopOrderDetailTool(shop),
		)
	}
	_ = log
	return tools
}

// ---------------------------------------------------------------------------
// Preferences
// ---------------------------------------------------------------------------

func readPreferencesTool(db *pgxpool.Pool) Tool {
	return newTool(
		"read_preferences",
		"Returns the family's cooking and shopping preferences as a markdown document. Read this once at the start of any planning session — preferences evolve.",
		map[string]any{},
		nil,
		func(ctx context.Context, _ json.RawMessage) (string, error) {
			rows, err := db.Query(ctx, `SELECT category, body_md FROM cooking_principles ORDER BY category`)
			if err != nil {
				return "", err
			}
			defer rows.Close()

			var buf strings.Builder
			for rows.Next() {
				var category, body string
				if err := rows.Scan(&category, &body); err != nil {
					return "", err
				}
				fmt.Fprintf(&buf, "## %s\n\n%s\n\n", category, strings.TrimSpace(body))
			}
			if err := rows.Err(); err != nil {
				return "", err
			}
			if buf.Len() == 0 {
				return "(no preferences recorded yet)", nil
			}
			return buf.String(), nil
		},
	)
}

func updatePreferenceTool(db *pgxpool.Pool) Tool {
	return newTool(
		"update_preference",
		"Replace the content of a preference category. Use for durable lessons learned (e.g. 'Noah doesn't like cilantro' belongs under family or dietary_constraints). Creating a new category is fine.",
		map[string]any{
			"category": map[string]any{
				"type":        "string",
				"description": "Slug-cased category name, e.g. cooking_style, shopping_rules, family.",
			},
			"body_md": map[string]any{
				"type":        "string",
				"description": "Full markdown body. This replaces any existing content under this category.",
			},
		},
		[]string{"category", "body_md"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Category string `json:"category"`
				Body     string `json:"body_md"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			tx, err := db.Begin(ctx)
			if err != nil {
				return "", err
			}
			defer func() { _ = tx.Rollback(ctx) }()

			if _, err := tx.Exec(ctx, `DELETE FROM cooking_principles WHERE category = $1`, in.Category); err != nil {
				return "", err
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO cooking_principles (category, body_md) VALUES ($1, $2)`,
				in.Category, in.Body); err != nil {
				return "", err
			}
			if err := tx.Commit(ctx); err != nil {
				return "", err
			}
			return fmt.Sprintf("updated %s (%d bytes)", in.Category, len(in.Body)), nil
		},
	)
}

// ---------------------------------------------------------------------------
// Dishes / weeks / dinners
// ---------------------------------------------------------------------------

func listDishesRecentTool(db *pgxpool.Pool) Tool {
	return newTool(
		"list_dishes_recent",
		"List dishes served in the last N weeks (default 4). Use this to avoid proposing the same dinner twice in a row and to prefer past hits.",
		map[string]any{
			"weeks": map[string]any{
				"type":        "integer",
				"description": "How many weeks back to consider.",
			},
		},
		nil,
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Weeks int `json:"weeks"`
			}
			_ = json.Unmarshal(input, &in)
			if in.Weeks <= 0 {
				in.Weeks = 4
			}
			rows, err := db.Query(ctx, `
				SELECT wd.day_date::text, d.name, d.cuisine, dr.rating, COALESCE(dr.notes, '')
				FROM week_dinners wd
				JOIN weeks w ON w.id = wd.week_id
				LEFT JOIN dishes d ON d.id = wd.dish_id
				LEFT JOIN dish_ratings dr ON dr.week_dinner_id = wd.id
				WHERE w.start_date >= current_date - ($1::int * interval '7 days')
				ORDER BY wd.day_date DESC
			`, in.Weeks)
			if err != nil {
				return "", err
			}
			defer rows.Close()
			var buf strings.Builder
			for rows.Next() {
				var day, ratingNotes string
				var name, cuisine, rating *string
				if err := rows.Scan(&day, &name, &cuisine, &rating, &ratingNotes); err != nil {
					return "", err
				}
				line := fmt.Sprintf("- %s: %s", day, coalesce(name, "(no dish linked)"))
				if cuisine != nil && *cuisine != "" {
					line += fmt.Sprintf(" (%s)", *cuisine)
				}
				if rating != nil && *rating != "" {
					line += fmt.Sprintf(" [%s]", *rating)
				}
				if ratingNotes != "" {
					line += fmt.Sprintf(" — %s", ratingNotes)
				}
				buf.WriteString(line)
				buf.WriteByte('\n')
			}
			if buf.Len() == 0 {
				return "(no dinners in the last " + fmt.Sprintf("%d", in.Weeks) + " weeks)", nil
			}
			return buf.String(), nil
		},
	)
}

func searchHistoryTool(db *pgxpool.Pool) Tool {
	return newTool(
		"search_history",
		"Search local history across all weeks. Returns past dinners (by dish name / cuisine / tags / recipe content) and past cart items (by product code / reason text) that match the query. Use when the user asks 'when did we last…', 'have we tried…', 'what did we pay for…', 'what kind of X do we usually buy'.",
		map[string]any{
			"query": map[string]any{"type": "string", "description": "Case-insensitive substring to match."},
			"kind": map[string]any{
				"type":        "string",
				"enum":        []string{"dishes", "cart", "both"},
				"description": "What to search (default 'both').",
			},
			"limit": map[string]any{"type": "integer", "description": "Max results per kind (default 10)."},
		},
		[]string{"query"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Query string `json:"query"`
				Kind  string `json:"kind"`
				Limit int    `json:"limit"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			q := strings.TrimSpace(in.Query)
			if q == "" {
				return "", fmt.Errorf("query required")
			}
			if in.Limit <= 0 {
				in.Limit = 10
			}
			kind := in.Kind
			if kind == "" {
				kind = "both"
			}
			// Escape % and _ so they aren't interpreted as ILIKE wildcards.
			// Backslash stays the escape character (default in Postgres).
			pat := "%" + escapeILIKE(q) + "%"
			var buf strings.Builder

			if kind == "dishes" || kind == "both" {
				rows, err := db.Query(ctx, `
					SELECT DISTINCT ON (d.id)
					       d.name, COALESCE(d.cuisine, ''),
					       wd.day_date::text, w.iso_week
					FROM dishes d
					JOIN week_dinners wd ON wd.dish_id = d.id
					JOIN weeks w ON w.id = wd.week_id
					WHERE d.name ILIKE $1
					   OR COALESCE(d.cuisine,'') ILIKE $1
					   OR d.recipe_md ILIKE $1
					   OR d.tags_json::text ILIKE $1
					ORDER BY d.id, wd.day_date DESC
					LIMIT $2`, pat, in.Limit)
				if err != nil {
					return "", err
				}
				var dinners []string
				for rows.Next() {
					var name, cuisine, day, iso string
					if err := rows.Scan(&name, &cuisine, &day, &iso); err != nil {
						rows.Close()
						return "", err
					}
					cuisineStr := ""
					if cuisine != "" {
						cuisineStr = " · " + cuisine
					}
					dinners = append(dinners, fmt.Sprintf("- %s (%s%s) — %s", name, day, cuisineStr, iso))
				}
				rows.Close()
				if len(dinners) > 0 {
					fmt.Fprintf(&buf, "## Dinners (%d)\n", len(dinners))
					for _, d := range dinners {
						buf.WriteString(d)
						buf.WriteByte('\n')
					}
				}
			}

			if kind == "cart" || kind == "both" {
				rows, err := db.Query(ctx, `
					SELECT ci.product_code, ci.qty, ci.reason_md, w.iso_week, w.start_date::text
					FROM cart_items ci
					JOIN weeks w ON w.id = ci.week_id
					WHERE ci.product_code ILIKE $1 OR ci.reason_md ILIKE $1
					ORDER BY w.start_date DESC, ci.id
					LIMIT $2`, pat, in.Limit)
				if err != nil {
					return "", err
				}
				var items []string
				for rows.Next() {
					var code, reason, iso, start string
					var qty float64
					if err := rows.Scan(&code, &qty, &reason, &iso, &start); err != nil {
						rows.Close()
						return "", err
					}
					items = append(items, fmt.Sprintf("- %s qty=%g  %s  (%s, %s)", code, qty, reason, iso, start))
				}
				rows.Close()
				if len(items) > 0 {
					if buf.Len() > 0 {
						buf.WriteByte('\n')
					}
					fmt.Fprintf(&buf, "## Cart items (%d)\n", len(items))
					for _, it := range items {
						buf.WriteString(it)
						buf.WriteByte('\n')
					}
				}
			}

			if buf.Len() == 0 {
				return fmt.Sprintf("(no history match for %q)", q), nil
			}
			return buf.String(), nil
		},
	)
}

func listWeeksTool(db *pgxpool.Pool) Tool {
	return newTool(
		"list_weeks",
		"List recent weekly plans with status and key dates.",
		map[string]any{
			"limit": map[string]any{"type": "integer", "description": "How many weeks (default 8)."},
		},
		nil,
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Limit int `json:"limit"`
			}
			_ = json.Unmarshal(input, &in)
			if in.Limit <= 0 {
				in.Limit = 8
			}
			rows, err := db.Query(ctx, `
				SELECT id, iso_week, start_date::text, end_date::text,
				       COALESCE(delivery_date::text, ''), status
				FROM weeks ORDER BY start_date DESC LIMIT $1`, in.Limit)
			if err != nil {
				return "", err
			}
			defer rows.Close()
			var buf strings.Builder
			for rows.Next() {
				var id int64
				var iso, status, delivery string
				var start, end string
				if err := rows.Scan(&id, &iso, &start, &end, &delivery, &status); err != nil {
					return "", err
				}
				line := fmt.Sprintf("- id=%d %s (%s → %s) [%s]", id, iso, start, end, status)
				if delivery != "" {
					line += fmt.Sprintf(" delivery=%s", delivery)
				}
				buf.WriteString(line)
				buf.WriteByte('\n')
			}
			if buf.Len() == 0 {
				return "(no weeks yet)", nil
			}
			return buf.String(), nil
		},
	)
}

func getWeekTool(db *pgxpool.Pool) Tool {
	return newTool(
		"get_week",
		"Fetch a week by id or iso_week (e.g. '2026-W17'), including its dinners, exceptions, and retrospective.",
		map[string]any{
			"id":       map[string]any{"type": "integer", "description": "Week id."},
			"iso_week": map[string]any{"type": "string", "description": "ISO week like 2026-W17."},
		},
		nil,
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				ID      int64  `json:"id"`
				IsoWeek string `json:"iso_week"`
			}
			_ = json.Unmarshal(input, &in)

			var weekID int64
			var iso, status, notes, start, end, delivery, order string
			var row pgx.Row
			const weekCols = `id, iso_week, start_date::text, end_date::text,
				COALESCE(delivery_date::text,''), COALESCE(order_date::text,''),
				status, notes_md`
			if in.ID > 0 {
				row = db.QueryRow(ctx, `SELECT `+weekCols+` FROM weeks WHERE id=$1`, in.ID)
			} else if in.IsoWeek != "" {
				row = db.QueryRow(ctx, `SELECT `+weekCols+` FROM weeks WHERE iso_week=$1`, in.IsoWeek)
			} else {
				return "", fmt.Errorf("need id or iso_week")
			}
			if err := row.Scan(&weekID, &iso, &start, &end, &delivery, &order, &status, &notes); err != nil {
				if err == pgx.ErrNoRows {
					return "(not found)", nil
				}
				return "", err
			}

			var buf strings.Builder
			fmt.Fprintf(&buf, "# Week %s (id=%d)\n%s → %s, status=%s\n", iso, weekID, start, end, status)
			if delivery != "" {
				fmt.Fprintf(&buf, "Delivery: %s\n", delivery)
			}
			if order != "" {
				fmt.Fprintf(&buf, "Order placed: %s\n", order)
			}
			if notes != "" {
				fmt.Fprintf(&buf, "\nNotes: %s\n", notes)
			}

			rows, err := db.Query(ctx, `
				SELECT wd.id, wd.day_date::text, wd.servings, wd.sourcing_json::text,
				       COALESCE(d.name,''), COALESCE(d.recipe_md,''), wd.notes,
				       dr.rating, COALESCE(dr.notes, '')
				FROM week_dinners wd
				LEFT JOIN dishes d ON d.id = wd.dish_id
				LEFT JOIN dish_ratings dr ON dr.week_dinner_id = wd.id
				WHERE wd.week_id = $1 ORDER BY wd.day_date, wd.sort_order
			`, weekID)
			if err != nil {
				return "", err
			}
			defer rows.Close()
			fmt.Fprintf(&buf, "\n## Dinners\n")
			for rows.Next() {
				var id int64
				var day, sourcing, name, recipe, dinnerNotes, ratingNotes string
				var rating *string
				var servings int
				if err := rows.Scan(&id, &day, &servings, &sourcing, &name, &recipe, &dinnerNotes,
					&rating, &ratingNotes); err != nil {
					return "", err
				}
				fmt.Fprintf(&buf, "\n### %s — %s (id=%d, %d pers)\n", day, name, id, servings)
				if sourcing != "" && sourcing != "{}" {
					fmt.Fprintf(&buf, "Sourcing: %s\n", sourcing)
				}
				if dinnerNotes != "" {
					fmt.Fprintf(&buf, "Notes: %s\n", dinnerNotes)
				}
				if rating != nil && *rating != "" {
					if ratingNotes != "" {
						fmt.Fprintf(&buf, "Verdict: %s — %s\n", *rating, ratingNotes)
					} else {
						fmt.Fprintf(&buf, "Verdict: %s\n", *rating)
					}
				}
				if recipe != "" {
					fmt.Fprintf(&buf, "\n%s\n", recipe)
				}
			}

			excRows, err := db.Query(ctx, `SELECT kind, description FROM week_exceptions WHERE week_id=$1`, weekID)
			if err == nil {
				defer excRows.Close()
				first := true
				for excRows.Next() {
					if first {
						fmt.Fprintf(&buf, "\n## Exceptions\n")
						first = false
					}
					var kind, desc string
					if err := excRows.Scan(&kind, &desc); err == nil {
						fmt.Fprintf(&buf, "- %s: %s\n", kind, desc)
					}
				}
			}

			var retroNotes string
			if err := db.QueryRow(ctx, `SELECT notes_md FROM retrospectives WHERE week_id=$1 ORDER BY created_at DESC LIMIT 1`, weekID).Scan(&retroNotes); err == nil {
				fmt.Fprintf(&buf, "\n## Retrospective\n\n%s\n", retroNotes)
			}

			cartRows, err := db.Query(ctx, `
				SELECT product_code, qty, reason_md
				FROM cart_items WHERE week_id = $1 ORDER BY added_at, id`, weekID)
			if err == nil {
				defer cartRows.Close()
				type ci struct {
					code, reason string
					qty          float64
				}
				var items []ci
				for cartRows.Next() {
					var c ci
					if err := cartRows.Scan(&c.code, &c.qty, &c.reason); err == nil {
						items = append(items, c)
					}
				}
				if len(items) > 0 {
					fmt.Fprintf(&buf, "\n## Cart (%d items)\n", len(items))
					for _, c := range items {
						fmt.Fprintf(&buf, "- %s  qty=%g  %s\n", c.code, c.qty, c.reason)
					}
				}
			}

			return buf.String(), nil
		},
	)
}

func createWeekTool(db *pgxpool.Pool) Tool {
	return newTool(
		"create_week",
		"Create a new weekly plan. Define it by start_date and end_date. iso_week is the label. delivery_date and order_date are optional post-hoc metadata — omit them unless the user has already placed the order.",
		map[string]any{
			"iso_week":      map[string]any{"type": "string", "description": "ISO week like 2026-W17. Compute from start_date if unsure."},
			"start_date":    map[string]any{"type": "string", "description": "YYYY-MM-DD, first menu day."},
			"end_date":      map[string]any{"type": "string", "description": "YYYY-MM-DD, last menu day. Usually start_date + 6."},
			"delivery_date": map[string]any{"type": "string", "description": "Optional metadata: YYYY-MM-DD the order is delivered. Set after ordering, not during planning."},
			"order_date":    map[string]any{"type": "string", "description": "Optional metadata: YYYY-MM-DD the order was placed on willys.se. Set after ordering."},
			"notes_md":      map[string]any{"type": "string", "description": "Optional week-level notes (e.g. 'Noah away Thu–Sun', 'fika bake for work Friday')."},
		},
		[]string{"iso_week", "start_date", "end_date"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				IsoWeek      string `json:"iso_week"`
				StartDate    string `json:"start_date"`
				EndDate      string `json:"end_date"`
				DeliveryDate string `json:"delivery_date"`
				OrderDate    string `json:"order_date"`
				Notes        string `json:"notes_md"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			var id int64
			err := db.QueryRow(ctx, `
				INSERT INTO weeks (iso_week, start_date, end_date, delivery_date, order_date, notes_md)
				VALUES ($1, $2::date, $3::date, NULLIF($4,'')::date, NULLIF($5,'')::date, $6)
				RETURNING id`,
				in.IsoWeek, in.StartDate, in.EndDate, in.DeliveryDate, in.OrderDate, in.Notes).Scan(&id)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("created week id=%d (%s, %s → %s)", id, in.IsoWeek, in.StartDate, in.EndDate), nil
		},
	)
}

func updateWeekTool(db *pgxpool.Pool) Tool {
	return newTool(
		"update_week",
		"Update the shape or metadata of an existing week. All fields are optional — pass only what changes. Look up by week_id or iso_week. Use this when the user wants to shift delivery, extend/shorten the menu, rename the week, change status, or edit notes.",
		map[string]any{
			"week_id":       map[string]any{"type": "integer", "description": "Preferred lookup."},
			"iso_week":      map[string]any{"type": "string", "description": "Alternative lookup (e.g. 2026-W17)."},
			"new_iso_week":  map[string]any{"type": "string", "description": "Rename the week label."},
			"start_date":    map[string]any{"type": "string", "description": "YYYY-MM-DD"},
			"end_date":      map[string]any{"type": "string", "description": "YYYY-MM-DD"},
			"delivery_date": map[string]any{"type": "string", "description": "YYYY-MM-DD. Empty string clears it."},
			"order_date":    map[string]any{"type": "string", "description": "YYYY-MM-DD. Empty string clears it."},
			"status":        map[string]any{"type": "string", "enum": []string{"draft", "cart_built", "ordered"}},
			"notes_md":      map[string]any{"type": "string"},
		},
		nil,
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				WeekID       int64   `json:"week_id"`
				IsoWeek      string  `json:"iso_week"`
				NewIsoWeek   *string `json:"new_iso_week"`
				StartDate    *string `json:"start_date"`
				EndDate      *string `json:"end_date"`
				DeliveryDate *string `json:"delivery_date"`
				OrderDate    *string `json:"order_date"`
				Status       *string `json:"status"`
				NotesMD      *string `json:"notes_md"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}

			id := in.WeekID
			if id == 0 {
				if in.IsoWeek == "" {
					return "", fmt.Errorf("need week_id or iso_week")
				}
				err := db.QueryRow(ctx, `SELECT id FROM weeks WHERE iso_week = $1`, in.IsoWeek).Scan(&id)
				if err != nil {
					if err == pgx.ErrNoRows {
						return fmt.Sprintf("no week with iso_week=%s", in.IsoWeek), nil
					}
					return "", err
				}
			}

			u := store.WeekUpdate{
				IsoWeek:      in.NewIsoWeek,
				StartDate:    in.StartDate,
				EndDate:      in.EndDate,
				DeliveryDate: in.DeliveryDate,
				OrderDate:    in.OrderDate,
				Status:       in.Status,
				NotesMD:      in.NotesMD,
			}
			n, err := store.UpdateWeek(ctx, db, id, u)
			if err != nil {
				return "", err
			}
			if n == 0 {
				return fmt.Sprintf("no week with id=%d", id), nil
			}
			return fmt.Sprintf("updated week id=%d", id), nil
		},
	)
}

func readHouseholdSettingsTool(db *pgxpool.Pool) Tool {
	return newTool(
		"read_household_settings",
		"Returns household-wide defaults: typical delivery weekday, number of dinners, order-to-delivery offset, default servings. Use these when planning a new week without explicit dates.",
		map[string]any{},
		nil,
		func(ctx context.Context, _ json.RawMessage) (string, error) {
			s, err := store.GetHouseholdSettings(ctx, db)
			if err != nil {
				return "", err
			}
			b, _ := json.MarshalIndent(s, "", "  ")
			return string(b), nil
		},
	)
}

func updateHouseholdSettingsTool(db *pgxpool.Pool) Tool {
	return newTool(
		"update_household_settings",
		"Update the household defaults. All fields optional — pass only what changes. Use when the user says 'we always order on Sundays' or 'make the default 6 dinners'.",
		map[string]any{
			"default_dinners":           map[string]any{"type": "integer", "description": "Typical number of dinners per week (1–14)."},
			"default_delivery_weekday":  map[string]any{"type": "integer", "description": "ISO weekday: 1=Mon, 2=Tue, … 7=Sun."},
			"default_order_offset_days": map[string]any{"type": "integer", "description": "Days between order and delivery. -1 = order the day before delivery."},
			"default_servings":          map[string]any{"type": "integer", "description": "Default dinner portions."},
			"notes_md":                  map[string]any{"type": "string", "description": "Free-form household notes."},
		},
		nil,
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				DefaultDinners         *int    `json:"default_dinners"`
				DefaultDeliveryWeekday *int    `json:"default_delivery_weekday"`
				DefaultOrderOffsetDays *int    `json:"default_order_offset_days"`
				DefaultServings        *int    `json:"default_servings"`
				NotesMD                *string `json:"notes_md"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			err := store.UpdateHouseholdSettings(ctx, db, store.HouseholdSettingsUpdate{
				DefaultDinners:         in.DefaultDinners,
				DefaultDeliveryWeekday: in.DefaultDeliveryWeekday,
				DefaultOrderOffsetDays: in.DefaultOrderOffsetDays,
				DefaultServings:        in.DefaultServings,
				NotesMD:                in.NotesMD,
			})
			if err != nil {
				return "", err
			}
			return "updated household settings", nil
		},
	)
}

func addDinnerTool(db *pgxpool.Pool) Tool {
	return newTool(
		"add_dinner",
		"Add a dinner to a week. Creates a dish row (if needed) and schedules it. Recipe goes in recipe_md as full markdown (ingredients + numbered steps).",
		map[string]any{
			"week_id":       map[string]any{"type": "integer"},
			"day_date":      map[string]any{"type": "string", "description": "YYYY-MM-DD"},
			"dish_name":     map[string]any{"type": "string"},
			"cuisine":       map[string]any{"type": "string"},
			"recipe_md":     map[string]any{"type": "string", "description": "Full recipe: ingredients list, numbered steps, technique notes."},
			"servings":      map[string]any{"type": "integer"},
			"sourcing_json": map[string]any{"type": "object", "description": "Optional mapping of source → item description, e.g. {\"butcher\": \"entrecote 4 pcs\", \"fishmonger\": \"sej 500g\"}", "additionalProperties": true},
			"notes":         map[string]any{"type": "string"},
			"tags":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Free tags like 'fish', 'weekend-easy', 'korean'."},
		},
		[]string{"week_id", "day_date", "dish_name"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				WeekID       int64           `json:"week_id"`
				DayDate      string          `json:"day_date"`
				DishName     string          `json:"dish_name"`
				Cuisine      string          `json:"cuisine"`
				RecipeMD     string          `json:"recipe_md"`
				Servings     int             `json:"servings"`
				SourcingJSON json.RawMessage `json:"sourcing_json"`
				Notes        string          `json:"notes"`
				Tags         []string        `json:"tags"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			if in.Servings == 0 {
				in.Servings = 4
			}
			sourcing := in.SourcingJSON
			if len(sourcing) == 0 {
				sourcing = json.RawMessage(`{}`)
			}
			tagsJSON, _ := json.Marshal(in.Tags)
			if len(in.Tags) == 0 {
				tagsJSON = []byte(`[]`)
			}

			tx, err := db.Begin(ctx)
			if err != nil {
				return "", err
			}
			defer func() { _ = tx.Rollback(ctx) }()

			var dishID int64
			err = tx.QueryRow(ctx, `
				INSERT INTO dishes (name, cuisine, recipe_md, servings, tags_json, last_made_at)
				VALUES ($1, NULLIF($2,''), $3, $4, $5::jsonb, $6::date)
				RETURNING id`,
				in.DishName, in.Cuisine, in.RecipeMD, in.Servings, string(tagsJSON), in.DayDate).Scan(&dishID)
			if err != nil {
				return "", err
			}
			var dinnerID int64
			err = tx.QueryRow(ctx, `
				INSERT INTO week_dinners (week_id, day_date, dish_id, servings, sourcing_json, notes)
				VALUES ($1, $2::date, $3, $4, $5::jsonb, $6)
				RETURNING id`,
				in.WeekID, in.DayDate, dishID, in.Servings, string(sourcing), in.Notes).Scan(&dinnerID)
			if err != nil {
				return "", err
			}
			if err := tx.Commit(ctx); err != nil {
				return "", err
			}
			return fmt.Sprintf("added dinner id=%d (dish_id=%d, '%s' on %s)", dinnerID, dishID, in.DishName, in.DayDate), nil
		},
	)
}

func updateDinnerTool(db *pgxpool.Pool) Tool {
	return newTool(
		"update_dinner",
		"Replace the dish attached to a dinner slot. Creates a new dish row and repoints the dinner; leaves the original dish row intact so history is preserved.",
		map[string]any{
			"dinner_id":     map[string]any{"type": "integer"},
			"dish_name":     map[string]any{"type": "string"},
			"cuisine":       map[string]any{"type": "string"},
			"recipe_md":     map[string]any{"type": "string"},
			"servings":      map[string]any{"type": "integer"},
			"sourcing_json": map[string]any{"type": "object", "additionalProperties": true},
			"notes":         map[string]any{"type": "string"},
		},
		[]string{"dinner_id", "dish_name", "recipe_md"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				DinnerID     int64           `json:"dinner_id"`
				DishName     string          `json:"dish_name"`
				Cuisine      string          `json:"cuisine"`
				RecipeMD     string          `json:"recipe_md"`
				Servings     int             `json:"servings"`
				SourcingJSON json.RawMessage `json:"sourcing_json"`
				Notes        string          `json:"notes"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			if in.Servings == 0 {
				in.Servings = 4
			}
			sourcing := in.SourcingJSON
			if len(sourcing) == 0 {
				sourcing = json.RawMessage(`{}`)
			}

			tx, err := db.Begin(ctx)
			if err != nil {
				return "", err
			}
			defer func() { _ = tx.Rollback(ctx) }()

			var dayDate string
			err = tx.QueryRow(ctx, `SELECT day_date::text FROM week_dinners WHERE id=$1`, in.DinnerID).Scan(&dayDate)
			if err != nil {
				return "", err
			}

			var dishID int64
			err = tx.QueryRow(ctx, `
				INSERT INTO dishes (name, cuisine, recipe_md, servings, last_made_at)
				VALUES ($1, NULLIF($2,''), $3, $4, $5::date)
				RETURNING id`,
				in.DishName, in.Cuisine, in.RecipeMD, in.Servings, dayDate).Scan(&dishID)
			if err != nil {
				return "", err
			}
			_, err = tx.Exec(ctx, `
				UPDATE week_dinners
				SET dish_id=$1, servings=$2, sourcing_json=$3::jsonb, notes=$4
				WHERE id=$5`,
				dishID, in.Servings, string(sourcing), in.Notes, in.DinnerID)
			if err != nil {
				return "", err
			}
			if err := tx.Commit(ctx); err != nil {
				return "", err
			}
			return fmt.Sprintf("updated dinner id=%d → dish '%s' (dish_id=%d)", in.DinnerID, in.DishName, dishID), nil
		},
	)
}

func deleteDinnerTool(db *pgxpool.Pool) Tool {
	return newTool(
		"delete_dinner",
		"Remove a scheduled dinner. The underlying dish row is kept for history.",
		map[string]any{
			"dinner_id": map[string]any{"type": "integer"},
		},
		[]string{"dinner_id"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				DinnerID int64 `json:"dinner_id"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			tag, err := db.Exec(ctx, `DELETE FROM week_dinners WHERE id=$1`, in.DinnerID)
			if err != nil {
				return "", err
			}
			if tag.RowsAffected() == 0 {
				return fmt.Sprintf("no dinner with id=%d", in.DinnerID), nil
			}
			return fmt.Sprintf("deleted dinner id=%d", in.DinnerID), nil
		},
	)
}

func addExceptionTool(db *pgxpool.Pool) Tool {
	return newTool(
		"add_exception",
		"Record a week-level exception (e.g. 'Noah away Thu–Sun' or 'extra fika bake Friday'). These inform future planning and appear on the week view.",
		map[string]any{
			"week_id":     map[string]any{"type": "integer"},
			"kind":        map[string]any{"type": "string", "description": "Short slug: 'absence', 'extra_meal', 'bake', 'other'."},
			"description": map[string]any{"type": "string"},
		},
		[]string{"week_id", "kind", "description"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				WeekID      int64  `json:"week_id"`
				Kind        string `json:"kind"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			var id int64
			err := db.QueryRow(ctx,
				`INSERT INTO week_exceptions (week_id, kind, description) VALUES ($1, $2, $3) RETURNING id`,
				in.WeekID, in.Kind, in.Description).Scan(&id)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("added exception id=%d (%s: %s)", id, in.Kind, in.Description), nil
		},
	)
}

func recordRetrospectiveTool(db *pgxpool.Pool) Tool {
	return newTool(
		"record_retrospective",
		"Save post-week feedback for a week. Use notes_md for free-form reflections. If the user mentioned specific dinners, include ratings — each rating references a week_dinner_id from get_week.",
		map[string]any{
			"week_id":  map[string]any{"type": "integer"},
			"notes_md": map[string]any{"type": "string"},
			"ratings": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"week_dinner_id": map[string]any{"type": "integer"},
						"rating":         map[string]any{"type": "string", "enum": []string{"loved", "liked", "meh", "disliked"}},
						"notes":          map[string]any{"type": "string"},
					},
					"required":             []string{"week_dinner_id", "rating"},
					"additionalProperties": false,
				},
			},
		},
		[]string{"week_id", "notes_md"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				WeekID  int64  `json:"week_id"`
				Notes   string `json:"notes_md"`
				Ratings []struct {
					DinnerID int64  `json:"week_dinner_id"`
					Rating   string `json:"rating"`
					Notes    string `json:"notes"`
				} `json:"ratings"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			tx, err := db.Begin(ctx)
			if err != nil {
				return "", err
			}
			defer func() { _ = tx.Rollback(ctx) }()

			if _, err := tx.Exec(ctx,
				`INSERT INTO retrospectives (week_id, notes_md) VALUES ($1, $2)`,
				in.WeekID, in.Notes); err != nil {
				return "", err
			}
			for _, r := range in.Ratings {
				if _, err := tx.Exec(ctx,
					`INSERT INTO dish_ratings (week_dinner_id, rating, notes)
					 VALUES ($1, $2, $3)
					 ON CONFLICT (week_dinner_id) DO UPDATE
					 SET rating = EXCLUDED.rating, notes = EXCLUDED.notes`,
					r.DinnerID, r.Rating, r.Notes); err != nil {
					return "", err
				}
			}
			if err := tx.Commit(ctx); err != nil {
				return "", err
			}
			return fmt.Sprintf("recorded retrospective for week %d with %d ratings", in.WeekID, len(in.Ratings)), nil
		},
	)
}

// ---------------------------------------------------------------------------
// Shopping provider (Willys today)
// ---------------------------------------------------------------------------

func shopSearchTool(shop shopping.Provider) Tool {
	return newTool(
		"willys_search",
		"Search the shopping provider's product catalog. Returns code, name, price, and per-unit price. Use this to map an ingredient to a specific product code before adding it to the cart.",
		map[string]any{
			"query": map[string]any{"type": "string", "description": "Product search term (Swedish works best on Willys)."},
			"limit": map[string]any{"type": "integer", "description": "Max results (default 10)."},
		},
		[]string{"query"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			products, err := shop.Search(ctx, in.Query, in.Limit)
			if err != nil {
				return "", err
			}
			var buf strings.Builder
			for _, p := range products {
				var tags []string
				if p.OnPromotion {
					tags = append(tags, "deal")
				}
				if p.Related {
					tags = append(tags, "related")
				}
				suffix := ""
				if len(tags) > 0 {
					suffix = " [" + strings.Join(tags, ",") + "]"
				}
				fmt.Fprintf(&buf, "%s\t%.2f kr\t%s\t%s%s\n",
					p.Code, p.Price, p.PricePerUnit, p.Name, suffix)
			}
			if buf.Len() == 0 {
				return fmt.Sprintf("(no results for %q)", in.Query), nil
			}
			return buf.String(), nil
		},
	)
}

func shopCartGetTool(shop shopping.Provider) Tool {
	return newTool(
		"willys_cart_get",
		"Return the current contents of the Willys cart and the total price in kronor.",
		map[string]any{},
		nil,
		func(ctx context.Context, _ json.RawMessage) (string, error) {
			cart, err := shop.CartGet(ctx)
			if err != nil {
				return "", err
			}
			return formatCart(cart), nil
		},
	)
}

func shopCartAddTool(shop shopping.Provider, db *pgxpool.Pool) Tool {
	return newTool(
		"willys_cart_add",
		"Add a product to the Willys cart by code (the value returned by willys_search). qty defaults to 1. When the user is planning a specific week (visible in chat context), pass reason so the line is saved to the week's shopping list. Returns the full updated cart.",
		map[string]any{
			"code":   map[string]any{"type": "string", "description": "Product code, e.g. '101253219_KG'."},
			"qty":    map[string]any{"type": "integer", "description": "Units to add (default 1)."},
			"reason": map[string]any{"type": "string", "description": "Why this product is in the cart. One short line. Example: 'Sej för pankopanerad sej (tors)'."},
		},
		[]string{"code"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Code   string `json:"code"`
				Qty    int    `json:"qty"`
				Reason string `json:"reason"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			qty := in.Qty
			if qty <= 0 {
				qty = 1
			}
			cart, err := shop.CartAdd(ctx, in.Code, qty)
			if err != nil {
				return "", err
			}
			if weekID := WeekIDFrom(ctx); weekID > 0 {
				snapshot := snapshotLine(cart, in.Code)
				_, _ = db.Exec(ctx, `
					INSERT INTO cart_items (week_id, product_code, qty, reason_md, committed, product_snapshot_json)
					VALUES ($1, $2, $3, $4, false, $5::jsonb)`,
					weekID, in.Code, qty, in.Reason, snapshot)
			}
			return fmt.Sprintf("added; cart now:\n%s", formatCart(cart)), nil
		},
	)
}

func shopCartAddManyTool(shop shopping.Provider, db *pgxpool.Pool) Tool {
	return newTool(
		"willys_cart_add_many",
		"Add many products to the Willys cart in one call. Prefer this over looping willys_cart_add. One iteration fits a full week's shop (40-60 items). Give each item a short reason line; when the chat is about a specific week, every reason gets saved to that week's shopping list alongside the product.",
		map[string]any{
			"items": map[string]any{
				"type":        "array",
				"description": "Products to add, in the order you want them tried.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code":   map[string]any{"type": "string", "description": "Product code from willys_search."},
						"qty":    map[string]any{"type": "integer", "description": "Units to add (default 1)."},
						"reason": map[string]any{"type": "string", "description": "Why this product is in the cart. One short line."},
					},
					"required":             []string{"code"},
					"additionalProperties": false,
				},
			},
		},
		[]string{"items"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Items []struct {
					Code   string `json:"code"`
					Qty    int    `json:"qty"`
					Reason string `json:"reason"`
				} `json:"items"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			weekID := WeekIDFrom(ctx)
			var success, failure int
			var errLines []string
			type recorded struct {
				code, reason string
				qty          int
			}
			toRecord := make([]recorded, 0, len(in.Items))
			for _, it := range in.Items {
				qty := it.Qty
				if qty <= 0 {
					qty = 1
				}
				if _, err := shop.CartAdd(ctx, it.Code, qty); err != nil {
					failure++
					errLines = append(errLines, fmt.Sprintf("%s: %v", it.Code, err))
					continue
				}
				success++
				if weekID > 0 {
					toRecord = append(toRecord, recorded{code: it.Code, reason: it.Reason, qty: qty})
				}
			}
			cart, err := shop.CartGet(ctx)
			if err != nil {
				return "", err
			}
			for _, r := range toRecord {
				snapshot := snapshotLine(cart, r.code)
				_, _ = db.Exec(ctx, `
					INSERT INTO cart_items (week_id, product_code, qty, reason_md, committed, product_snapshot_json)
					VALUES ($1, $2, $3, $4, false, $5::jsonb)`,
					weekID, r.code, r.qty, r.reason, snapshot)
			}
			var buf strings.Builder
			fmt.Fprintf(&buf, "added %d/%d items\n", success, success+failure)
			if len(errLines) > 0 {
				buf.WriteString("\nerrors:\n")
				for _, l := range errLines {
					buf.WriteString(l)
					buf.WriteByte('\n')
				}
			}
			buf.WriteString("\ncart now:\n")
			buf.WriteString(formatCart(cart))
			return buf.String(), nil
		},
	)
}

func shopCartRemoveTool(shop shopping.Provider, db *pgxpool.Pool) Tool {
	return newTool(
		"willys_cart_remove",
		"Remove a product from the Willys cart by code. Also drops matching rows from the week's shopping list when chat is in-week.",
		map[string]any{
			"code": map[string]any{"type": "string"},
		},
		[]string{"code"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			cart, err := shop.CartRemove(ctx, in.Code)
			if err != nil {
				return "", err
			}
			if weekID := WeekIDFrom(ctx); weekID > 0 {
				_, _ = db.Exec(ctx,
					`DELETE FROM cart_items WHERE week_id = $1 AND product_code = $2`,
					weekID, in.Code)
			}
			return fmt.Sprintf("removed; cart now:\n%s", formatCart(cart)), nil
		},
	)
}

func shopCartClearTool(shop shopping.Provider, db *pgxpool.Pool) Tool {
	return newTool(
		"willys_cart_clear",
		"Empty the Willys cart. Use when the user confirms they want a fresh start. Also clears the current week's shopping list when chat is in-week.",
		map[string]any{},
		nil,
		func(ctx context.Context, _ json.RawMessage) (string, error) {
			if err := shop.CartClear(ctx); err != nil {
				return "", err
			}
			if weekID := WeekIDFrom(ctx); weekID > 0 {
				_, _ = db.Exec(ctx, `DELETE FROM cart_items WHERE week_id = $1`, weekID)
			}
			return "cart cleared", nil
		},
	)
}

func shopOrderDetailTool(shop shopping.Provider) Tool {
	return newTool(
		"willys_order_detail",
		"Fetch the full line items of a past Willys order by its id (from willys_orders_recent). Use this to check what was actually bought — quantities, brands, prices — when planning a new week to mirror or tweak a prior order.",
		map[string]any{
			"id": map[string]any{"type": "string", "description": "Order number from willys_orders_recent."},
		},
		[]string{"id"},
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", err
			}
			d, err := shop.OrderDetail(ctx, in.ID)
			if err != nil {
				return "", err
			}
			var buf strings.Builder
			fmt.Fprintf(&buf, "Order %s — %s — %.2f kr — %s\n\n",
				d.ID, d.Date, d.Total, d.Status)
			// Group by category for readability.
			byCat := map[string][]shopping.OrderLine{}
			var cats []string
			for _, l := range d.Lines {
				if _, ok := byCat[l.Category]; !ok {
					cats = append(cats, l.Category)
				}
				byCat[l.Category] = append(byCat[l.Category], l)
			}
			for _, cat := range cats {
				if cat != "" {
					fmt.Fprintf(&buf, "## %s\n", cat)
				}
				for _, l := range byCat[cat] {
					fmt.Fprintf(&buf, "%s\tqty=%.0f\t%.2f kr\t%s\n",
						l.Code, l.Qty, l.LineTotal, l.Name)
				}
				buf.WriteByte('\n')
			}
			return buf.String(), nil
		},
	)
}

func shopOrdersRecentTool(shop shopping.Provider) Tool {
	return newTool(
		"willys_orders_recent",
		"Return the most recent Willys orders (id, delivery date, total). Useful when planning to mirror a previous order or sanity-check seasonal produce availability.",
		map[string]any{
			"limit": map[string]any{"type": "integer", "description": "Number of orders (default 5)."},
		},
		nil,
		func(ctx context.Context, input json.RawMessage) (string, error) {
			var in struct {
				Limit int `json:"limit"`
			}
			_ = json.Unmarshal(input, &in)
			orders, err := shop.OrdersRecent(ctx, in.Limit)
			if err != nil {
				return "", err
			}
			if len(orders) == 0 {
				return "(no recent orders)", nil
			}
			var buf strings.Builder
			for _, o := range orders {
				fmt.Fprintf(&buf, "%s\t%s\t%.2f kr\n", o.ID, o.Date, o.Total)
			}
			return buf.String(), nil
		},
	)
}

func formatCart(c shopping.Cart) string {
	if len(c.Items) == 0 {
		return "(cart is empty)"
	}
	var buf strings.Builder
	for _, it := range c.Items {
		fmt.Fprintf(&buf, "%s\tqty=%.0f\t%.2f kr\t%s\n",
			it.Code, it.Qty, it.LineTotal, it.Name)
	}
	fmt.Fprintf(&buf, "\ntotal: %.2f kr", c.Total)
	return buf.String()
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func coalesce(s *string, def string) string {
	if s == nil || *s == "" {
		return def
	}
	return *s
}

// escapeILIKE escapes % and _ so a user query like "50%" doesn't match
// every row. Backslash is Postgres' default ILIKE escape character.
func escapeILIKE(q string) string {
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	return q
}

// snapshotLine returns the JSON blob to store alongside a cart_items row so
// the UI can render a name and a price without a live product lookup. Keys
// are deliberately minimal; add fields here when a new one earns its place
// rather than dumping the whole upstream shape.
func snapshotLine(cart shopping.Cart, code string) string {
	for _, l := range cart.Items {
		if l.Code == code {
			b, _ := json.Marshal(map[string]any{
				"name":       l.Name,
				"unit_price": l.UnitPrice,
				"line_total": l.LineTotal,
			})
			return string(b)
		}
	}
	return "{}"
}
