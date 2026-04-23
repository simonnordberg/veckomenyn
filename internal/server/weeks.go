package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/simonnordberg/veckomenyn/internal/store"
)

type weekSummary struct {
	ID           int64   `json:"id"`
	IsoWeek      string  `json:"iso_week"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	DeliveryDate *string `json:"delivery_date"`
	OrderDate    *string `json:"order_date"`
	Status       string  `json:"status"`
	DinnerCount  int     `json:"dinner_count"`
	UpdatedAt    string  `json:"updated_at"`
}

type dinnerOut struct {
	ID          int64           `json:"id"`
	DayDate     string          `json:"day_date"`
	DishID      *int64          `json:"dish_id"`
	DishName    string          `json:"dish_name"`
	Cuisine     *string         `json:"cuisine"`
	Servings    int             `json:"servings"`
	Sourcing    json.RawMessage `json:"sourcing"`
	RecipeMD    string          `json:"recipe_md"`
	Notes       string          `json:"notes"`
	Rating      *string         `json:"rating"`
	RatingNotes string          `json:"rating_notes"`
}

type exceptionOut struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type retrospectiveOut struct {
	ID        int64  `json:"id"`
	NotesMD   string `json:"notes_md"`
	CreatedAt string `json:"created_at"`
}

type cartItemOut struct {
	ID          int64           `json:"id"`
	ProductCode string          `json:"product_code"`
	Qty         float64         `json:"qty"`
	ReasonMD    string          `json:"reason_md"`
	Committed   bool            `json:"committed"`
	Snapshot    json.RawMessage `json:"snapshot"`
}

type weekDetail struct {
	weekSummary
	NotesMD        string             `json:"notes_md"`
	Dinners        []dinnerOut        `json:"dinners"`
	Exceptions     []exceptionOut     `json:"exceptions"`
	Retrospectives []retrospectiveOut `json:"retrospectives"`
	CartItems      []cartItemOut      `json:"cart_items"`
}

const weekSelect = `
	SELECT w.id, w.iso_week, w.start_date::text, w.end_date::text,
	       w.delivery_date::text, w.order_date::text, w.status,
	       COALESCE((SELECT COUNT(*) FROM week_dinners wd WHERE wd.week_id = w.id), 0) AS dinner_count,
	       w.updated_at::text
	FROM weeks w
`

func scanWeekSummary(row interface{ Scan(...any) error }) (weekSummary, error) {
	var w weekSummary
	var delivery, order sql.NullString
	err := row.Scan(&w.ID, &w.IsoWeek, &w.StartDate, &w.EndDate, &delivery, &order, &w.Status, &w.DinnerCount, &w.UpdatedAt)
	if err != nil {
		return w, err
	}
	if delivery.Valid {
		w.DeliveryDate = &delivery.String
	}
	if order.Valid {
		w.OrderDate = &order.String
	}
	return w, nil
}

func (s *Server) handleListWeeks(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Pool.Query(r.Context(), weekSelect+`ORDER BY w.start_date DESC NULLS LAST LIMIT 50`)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	defer rows.Close()
	out := []weekSummary{}
	for rows.Next() {
		ws, err := scanWeekSummary(rows)
		if err != nil {
			s.internalError(w, r, "request", err)
			return
		}
		out = append(out, ws)
	}
	writeJSON(w, http.StatusOK, map[string]any{"weeks": out})
}

func (s *Server) handleCurrentWeek(w http.ResponseWriter, r *http.Request) {
	// "Current" picks the week most likely to be relevant right now:
	//   1. one whose date range covers today,
	//   2. else the nearest upcoming week,
	//   3. else the most recent past week.
	// Returns 204 if no weeks exist at all.
	row := s.db.Pool.QueryRow(r.Context(), weekSelect+`
		ORDER BY
		  CASE
		    WHEN current_date BETWEEN w.start_date AND w.end_date THEN 0
		    WHEN w.start_date > current_date THEN 1
		    ELSE 2
		  END,
		  CASE
		    WHEN w.start_date > current_date THEN w.start_date - current_date
		    ELSE current_date - w.end_date
		  END
		LIMIT 1`)
	ws, err := scanWeekSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	detail, err := s.loadWeekDetail(r, ws)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// handleDeleteWeek removes a plan and everything that hangs off it. The FK
// constraints have ON DELETE CASCADE, so dinners, exceptions, cart items,
// retrospectives, and conversation links all go with it.
func (s *Server) handleDeleteWeek(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	tag, err := s.db.Pool.Exec(r.Context(), `DELETE FROM weeks WHERE id = $1`, id)
	if err != nil {
		s.internalError(w, r, "delete week", err)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetWeekByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	row := s.db.Pool.QueryRow(r.Context(), weekSelect+`WHERE w.id = $1`, id)
	ws, err := scanWeekSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	detail, err := s.loadWeekDetail(r, ws)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleGetWeek(w http.ResponseWriter, r *http.Request) {
	iso := chi.URLParam(r, "iso")
	if !isoWeekRE.MatchString(iso) {
		http.Error(w, "bad iso_week (expected YYYY-Www)", http.StatusBadRequest)
		return
	}
	// iso_week is a label, not an identity: multiple plans can share it.
	// Resolve to the most recently updated match so the URL points to the
	// one most likely being worked on.
	row := s.db.Pool.QueryRow(r.Context(),
		weekSelect+`WHERE w.iso_week = $1 ORDER BY w.updated_at DESC LIMIT 1`, iso)
	ws, err := scanWeekSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	detail, err := s.loadWeekDetail(r, ws)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

type weekCreate struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	NotesMD   string `json:"notes_md"`
}

// handleCreateWeek inserts a fresh draft plan from a start/end date pair.
// Deterministic: no agent involvement. The iso_week label is derived from
// start_date; the user can rename it later.
func (s *Server) handleCreateWeek(w http.ResponseWriter, r *http.Request) {
	var in weekCreate
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !dateRE.MatchString(in.StartDate) || !dateRE.MatchString(in.EndDate) {
		http.Error(w, "start_date and end_date must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	var id int64
	err := s.db.Pool.QueryRow(r.Context(), `
		INSERT INTO weeks (iso_week, start_date, end_date, notes_md)
		VALUES (to_char($1::date, 'IYYY-"W"IW'), $1::date, $2::date, $3)
		RETURNING id`, in.StartDate, in.EndDate, in.NotesMD).Scan(&id)
	if err != nil {
		s.internalError(w, r, "create week", err)
		return
	}

	row := s.db.Pool.QueryRow(r.Context(), weekSelect+`WHERE w.id = $1`, id)
	ws, err := scanWeekSummary(row)
	if err != nil {
		s.internalError(w, r, "create week reload", err)
		return
	}
	detail, err := s.loadWeekDetail(r, ws)
	if err != nil {
		s.internalError(w, r, "create week detail", err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

type weekPatch struct {
	IsoWeek      *string `json:"iso_week"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	DeliveryDate *string `json:"delivery_date"`
	OrderDate    *string `json:"order_date"`
	Status       *string `json:"status"`
	NotesMD      *string `json:"notes_md"`
}

// handlePatchWeek applies a partial update to a week and returns the fresh
// detail so the UI can update in one round trip.
//
// Date edits carry side-effects so the schedule stays consistent with
// what's inside the plan:
//   - Changing start_date shifts every dinner's day_date by the same delta
//     and slides end_date forward/back by the same amount (preserves the
//     plan's duration). iso_week is recomputed from the new start unless
//     the caller set it explicitly.
//   - Shrinking end_date deletes dinners whose day_date falls past the new
//     end. The UI asks for confirmation before sending that.
//   - Extending end_date inserts empty dinner rows for each new day so the
//     new slots surface in the UI.
func (s *Server) handlePatchWeek(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	var p weekPatch
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if p.StartDate != nil && !dateRE.MatchString(*p.StartDate) {
		http.Error(w, "start_date must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	if p.EndDate != nil && !dateRE.MatchString(*p.EndDate) {
		http.Error(w, "end_date must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "patch begin", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var curStart, curEnd string
	err = tx.QueryRow(ctx, `SELECT start_date::text, end_date::text FROM weeks WHERE id = $1`, id).Scan(&curStart, &curEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.internalError(w, r, "patch load", err)
		return
	}

	// Apply start-shift side-effects before the standard update runs.
	effectiveEnd := curEnd
	if p.StartDate != nil && *p.StartDate != curStart {
		if _, err := tx.Exec(ctx, `
			UPDATE week_dinners
			SET day_date = day_date + ($1::date - $2::date)
			WHERE week_id = $3`, *p.StartDate, curStart, id); err != nil {
			s.internalError(w, r, "patch shift dinners", err)
			return
		}
		// Slide end_date by the same delta unless the caller set it too.
		if p.EndDate == nil {
			if err := tx.QueryRow(ctx, `
				SELECT ($1::date + ($2::date - $3::date))::text`,
				curEnd, *p.StartDate, curStart).Scan(&effectiveEnd); err != nil {
				s.internalError(w, r, "patch compute end", err)
				return
			}
			p.EndDate = &effectiveEnd
		}
		// Keep the iso_week label aligned with start_date by default.
		if p.IsoWeek == nil {
			var newIso string
			if err := tx.QueryRow(ctx, `SELECT to_char($1::date, 'IYYY-"W"IW')`, *p.StartDate).Scan(&newIso); err != nil {
				s.internalError(w, r, "patch compute iso_week", err)
				return
			}
			p.IsoWeek = &newIso
		}
		// The standard update below will rewrite start_date too; update our
		// cached curStart so downstream range checks use the right baseline.
		curStart = *p.StartDate
		curEnd = effectiveEnd
	}

	// Apply end-resize side-effects.
	if p.EndDate != nil && *p.EndDate != curEnd {
		if *p.EndDate < curStart {
			http.Error(w, "end_date cannot be before start_date", http.StatusBadRequest)
			return
		}
		if *p.EndDate < curEnd {
			if _, err := tx.Exec(ctx, `
				DELETE FROM week_dinners
				WHERE week_id = $1 AND day_date > $2::date`, id, *p.EndDate); err != nil {
				s.internalError(w, r, "patch truncate dinners", err)
				return
			}
		} else if *p.EndDate > curEnd {
			// Insert one empty slot per added day so the new days surface in
			// the UI as (untitled) dinners, ready to be filled.
			if _, err := tx.Exec(ctx, `
				INSERT INTO week_dinners (week_id, day_date)
				SELECT $1, gs::date
				FROM generate_series(($2::date + 1), $3::date, '1 day') gs`,
				id, curEnd, *p.EndDate); err != nil {
				s.internalError(w, r, "patch add empty slots", err)
				return
			}
		}
	}

	n, err := store.UpdateWeek(ctx, tx, id, store.WeekUpdate{
		IsoWeek:      p.IsoWeek,
		StartDate:    p.StartDate,
		EndDate:      p.EndDate,
		DeliveryDate: p.DeliveryDate,
		OrderDate:    p.OrderDate,
		Status:       p.Status,
		NotesMD:      p.NotesMD,
	})
	if err != nil {
		s.internalError(w, r, "patch update week", err)
		return
	}
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		s.internalError(w, r, "patch commit", err)
		return
	}

	row := s.db.Pool.QueryRow(ctx, weekSelect+`WHERE w.id = $1`, id)
	ws, err := scanWeekSummary(row)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	detail, err := s.loadWeekDetail(r, ws)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) loadWeekDetail(r *http.Request, ws weekSummary) (*weekDetail, error) {
	ctx := r.Context()

	var notes string
	if err := s.db.Pool.QueryRow(ctx, `SELECT notes_md FROM weeks WHERE id=$1`, ws.ID).Scan(&notes); err != nil {
		return nil, err
	}

	dinners, err := s.loadDinners(r, ws.ID)
	if err != nil {
		return nil, err
	}
	exceptions, err := s.loadExceptions(r, ws.ID)
	if err != nil {
		return nil, err
	}
	retros, err := s.loadRetrospectives(r, ws.ID)
	if err != nil {
		return nil, err
	}
	cart, err := s.loadCartItems(r, ws.ID)
	if err != nil {
		return nil, err
	}

	return &weekDetail{
		weekSummary:    ws,
		NotesMD:        notes,
		Dinners:        dinners,
		Exceptions:     exceptions,
		Retrospectives: retros,
		CartItems:      cart,
	}, nil
}

func (s *Server) loadCartItems(r *http.Request, weekID int64) ([]cartItemOut, error) {
	rows, err := s.db.Pool.Query(r.Context(), `
		SELECT id, product_code, qty, reason_md, committed,
		       COALESCE(product_snapshot_json, '{}'::jsonb)::text
		FROM cart_items WHERE week_id = $1
		ORDER BY added_at, id`, weekID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []cartItemOut{}
	for rows.Next() {
		var c cartItemOut
		var snap string
		if err := rows.Scan(&c.ID, &c.ProductCode, &c.Qty, &c.ReasonMD, &c.Committed, &snap); err != nil {
			return nil, err
		}
		c.Snapshot = json.RawMessage(snap)
		out = append(out, c)
	}
	return out, nil
}

func (s *Server) loadDinners(r *http.Request, weekID int64) ([]dinnerOut, error) {
	rows, err := s.db.Pool.Query(r.Context(), `
		SELECT wd.id, wd.day_date::text, wd.dish_id, wd.servings,
		       COALESCE(wd.sourcing_json, '{}'::jsonb)::text,
		       COALESCE(d.name, ''), d.cuisine, COALESCE(d.recipe_md, ''), wd.notes,
		       dr.rating, COALESCE(dr.notes, '')
		FROM week_dinners wd
		LEFT JOIN dishes d ON d.id = wd.dish_id
		LEFT JOIN dish_ratings dr ON dr.week_dinner_id = wd.id
		WHERE wd.week_id = $1
		ORDER BY wd.day_date, wd.sort_order`, weekID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dinnerOut{}
	for rows.Next() {
		var d dinnerOut
		var dishID sql.NullInt64
		var cuisine, rating sql.NullString
		var sourcingRaw string
		if err := rows.Scan(&d.ID, &d.DayDate, &dishID, &d.Servings, &sourcingRaw,
			&d.DishName, &cuisine, &d.RecipeMD, &d.Notes,
			&rating, &d.RatingNotes); err != nil {
			return nil, err
		}
		if dishID.Valid {
			v := dishID.Int64
			d.DishID = &v
		}
		if cuisine.Valid {
			v := cuisine.String
			d.Cuisine = &v
		}
		if rating.Valid {
			v := rating.String
			d.Rating = &v
		}
		d.Sourcing = json.RawMessage(sourcingRaw)
		out = append(out, d)
	}
	return out, nil
}

func (s *Server) loadExceptions(r *http.Request, weekID int64) ([]exceptionOut, error) {
	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, kind, description FROM week_exceptions WHERE week_id=$1 ORDER BY id`, weekID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []exceptionOut{}
	for rows.Next() {
		var e exceptionOut
		if err := rows.Scan(&e.ID, &e.Kind, &e.Description); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

type retrospectivePut struct {
	NotesMD string `json:"notes_md"`
}

// handlePutWeekRetrospective upserts the week-level free-form retrospective
// (pacing, portions, general feedback for next week's planning).
func (s *Server) handlePutWeekRetrospective(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	var body retrospectivePut
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := store.UpsertWeekRetrospective(r.Context(), s.db.Pool, id, body.NotesMD); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		s.internalError(w, r, "upsert retrospective", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type cloneInput struct {
	StartDate string `json:"start_date"` // optional; defaults to source.start + 7
}

// handleCloneWeek forks any plan into a new draft. By default the new plan
// starts 7 days after the source; caller can pass a specific start_date in
// the body and the duration is preserved. Dinners are copied with their
// day_date shifted by the same offset; ratings, retrospectives, cart items,
// and week-level notes stay with the source. Multiple plans may share the
// same iso_week label: the new plan is always a fresh row.
func (s *Server) handleCloneWeek(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	var in cloneInput
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if in.StartDate != "" && !dateRE.MatchString(in.StartDate) {
		http.Error(w, "start_date must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "clone begin", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// NULLIF('','') lets the same query handle "default to +7" and an explicit
	// start_date. The dinner-shift offset is new_start - source_start, so the
	// new plan preserves the source's duration.
	var targetID int64
	err = tx.QueryRow(ctx, `
		WITH src AS (
		  SELECT start_date AS s, end_date AS e FROM weeks WHERE id = $2
		), target AS (
		  SELECT COALESCE(NULLIF($1,'')::date, s + 7) AS new_start,
		         e - s AS duration
		  FROM src
		)
		INSERT INTO weeks (iso_week, start_date, end_date)
		SELECT to_char(new_start, 'IYYY-"W"IW'), new_start, new_start + duration
		FROM target
		RETURNING id`, in.StartDate, sourceID).Scan(&targetID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.internalError(w, r, "clone create target", err)
		return
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO week_dinners (week_id, day_date, dish_id, servings, sourcing_json, notes, sort_order)
		SELECT $1,
		       day_date + ((SELECT start_date FROM weeks WHERE id = $1) - (SELECT start_date FROM weeks WHERE id = $2)),
		       dish_id, servings, sourcing_json, notes, sort_order
		FROM week_dinners WHERE week_id = $2`, targetID, sourceID); err != nil {
		s.internalError(w, r, "clone copy dinners", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		s.internalError(w, r, "clone commit", err)
		return
	}

	row := s.db.Pool.QueryRow(ctx, weekSelect+`WHERE w.id = $1`, targetID)
	ws, err := scanWeekSummary(row)
	if err != nil {
		s.internalError(w, r, "clone reload", err)
		return
	}
	detail, err := s.loadWeekDetail(r, ws)
	if err != nil {
		s.internalError(w, r, "clone detail", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) loadRetrospectives(r *http.Request, weekID int64) ([]retrospectiveOut, error) {
	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, notes_md, created_at::text FROM retrospectives WHERE week_id=$1 ORDER BY created_at DESC`, weekID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []retrospectiveOut{}
	for rows.Next() {
		var r retrospectiveOut
		if err := rows.Scan(&r.ID, &r.NotesMD, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
