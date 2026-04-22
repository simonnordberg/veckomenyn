package server

import (
	"encoding/json"
	"net/http"

	"github.com/simonnordberg/veckomenyn/internal/store"
)

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := store.GetHouseholdSettings(r.Context(), s.db.Pool)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

type settingsPatch struct {
	DefaultDinners         *int    `json:"default_dinners"`
	DefaultDeliveryWeekday *int    `json:"default_delivery_weekday"`
	DefaultOrderOffsetDays *int    `json:"default_order_offset_days"`
	DefaultServings        *int    `json:"default_servings"`
	Language               *string `json:"language"`
	NotesMD                *string `json:"notes_md"`
}

func (s *Server) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	var p settingsPatch
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	err := store.UpdateHouseholdSettings(r.Context(), s.db.Pool, store.HouseholdSettingsUpdate{
		DefaultDinners:         p.DefaultDinners,
		DefaultDeliveryWeekday: p.DefaultDeliveryWeekday,
		DefaultOrderOffsetDays: p.DefaultOrderOffsetDays,
		DefaultServings:        p.DefaultServings,
		Language:               p.Language,
		NotesMD:                p.NotesMD,
	})
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	settings, err := store.GetHouseholdSettings(r.Context(), s.db.Pool)
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}
