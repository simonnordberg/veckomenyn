package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

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
	ID        int64           `json:"id"`
	DayDate   string          `json:"day_date"`
	DishID    *int64          `json:"dish_id"`
	DishName  string          `json:"dish_name"`
	Cuisine   *string         `json:"cuisine"`
	Servings  int             `json:"servings"`
	Sourcing  json.RawMessage `json:"sourcing"`
	RecipeMD  string          `json:"recipe_md"`
	Notes     string          `json:"notes"`
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

func (s *Server) handleGetWeek(w http.ResponseWriter, r *http.Request) {
	iso := chi.URLParam(r, "iso")
	if !isoWeekRE.MatchString(iso) {
		http.Error(w, "bad iso_week (expected YYYY-Www)", http.StatusBadRequest)
		return
	}
	row := s.db.Pool.QueryRow(r.Context(), weekSelect+`WHERE w.iso_week = $1`, iso)
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

	n, err := store.UpdateWeek(r.Context(), s.db.Pool, id, store.WeekUpdate{
		IsoWeek:      p.IsoWeek,
		StartDate:    p.StartDate,
		EndDate:      p.EndDate,
		DeliveryDate: p.DeliveryDate,
		OrderDate:    p.OrderDate,
		Status:       p.Status,
		NotesMD:      p.NotesMD,
	})
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	row := s.db.Pool.QueryRow(r.Context(), weekSelect+`WHERE w.id = $1`, id)
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
		       COALESCE(d.name, ''), d.cuisine, COALESCE(d.recipe_md, ''), wd.notes
		FROM week_dinners wd
		LEFT JOIN dishes d ON d.id = wd.dish_id
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
		var cuisine sql.NullString
		var sourcingRaw string
		if err := rows.Scan(&d.ID, &d.DayDate, &dishID, &d.Servings, &sourcingRaw, &d.DishName, &cuisine, &d.RecipeMD, &d.Notes); err != nil {
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
