package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/simonnordberg/veckomenyn/internal/store"
)

type dinnerRatingPut struct {
	Rating string `json:"rating"`
	Notes  string `json:"notes"`
}

// handlePutDinnerRating upserts a rating on a dinner. One rating per dinner.
func (s *Server) handlePutDinnerRating(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	var body dinnerRatingPut
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !store.ValidDinnerRatings[body.Rating] {
		http.Error(w, "bad rating (expected one of loved/liked/meh/disliked)", http.StatusBadRequest)
		return
	}
	if err := store.SetDinnerRating(r.Context(), s.db.Pool, id, body.Rating, body.Notes); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		s.internalError(w, r, "set dinner rating", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rating":       body.Rating,
		"rating_notes": body.Notes,
	})
}

// handleDeleteDinnerRating clears the rating for a dinner, if any.
func (s *Server) handleDeleteDinnerRating(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r, "id")
	if !ok {
		return
	}
	if err := store.ClearDinnerRating(r.Context(), s.db.Pool, id); err != nil {
		s.internalError(w, r, "clear dinner rating", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
