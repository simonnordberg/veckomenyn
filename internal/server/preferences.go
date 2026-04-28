package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// Category slugs are user-controlled but must be URL-safe. Enforce a
// conservative shape so the REST surface is predictable.
var categoryRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func validCategory(s string) bool { return categoryRE.MatchString(s) }

// preferenceOut is one row of cooking_principles. Category is the stable slug;
// body_md is free-form markdown.
type preferenceOut struct {
	Category  string `json:"category"`
	BodyMD    string `json:"body_md"`
	UpdatedAt string `json:"updated_at"`
}

func (s *Server) handleListPreferences(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Pool.Query(r.Context(), `
		SELECT category, body_md, updated_at::text
		FROM cooking_principles
		ORDER BY category`)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	defer rows.Close()
	out := []preferenceOut{}
	for rows.Next() {
		var p preferenceOut
		if err := rows.Scan(&p.Category, &p.BodyMD, &p.UpdatedAt); err != nil {
			s.internalError(w, r, "request", err)
			return
		}
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"preferences": out})
}

type preferencePatch struct {
	BodyMD *string `json:"body_md"`
}

// handlePutPreference upserts a preference row. Treat the whole document as
// the unit; replace on write.
func (s *Server) handlePutPreference(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if !validCategory(category) {
		http.Error(w, "bad category (expected a-z0-9_- slug)", http.StatusBadRequest)
		return
	}
	var p preferencePatch
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if p.BodyMD == nil {
		http.Error(w, "body_md required", http.StatusBadRequest)
		return
	}

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	if _, err := tx.Exec(r.Context(),
		`DELETE FROM cooking_principles WHERE category = $1`, category); err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO cooking_principles (category, body_md) VALUES ($1, $2)`,
		category, *p.BodyMD); err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.internalError(w, r, "request", err)
		return
	}

	var out preferenceOut
	err = s.db.Pool.QueryRow(r.Context(), `
		SELECT category, body_md, updated_at::text
		FROM cooking_principles WHERE category = $1`, category).
		Scan(&out.Category, &out.BodyMD, &out.UpdatedAt)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDeletePreference(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if !validCategory(category) {
		http.Error(w, "bad category (expected a-z0-9_- slug)", http.StatusBadRequest)
		return
	}
	tag, err := s.db.Pool.Exec(r.Context(),
		`DELETE FROM cooking_principles WHERE category = $1`, category)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	if tag.RowsAffected() == 0 && !errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
