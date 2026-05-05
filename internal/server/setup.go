package server

import (
	"net/http"

	"github.com/simonnordberg/veckomenyn/internal/seed"
	"github.com/simonnordberg/veckomenyn/internal/store"
)

// setupStatusDTO drives the first-run wizard. setup_complete flips to true
// the moment an LLM provider has been selected and its config saved.
type setupStatusDTO struct {
	SetupComplete    bool `json:"setup_complete"`
	HasLLMProvider   bool `json:"has_llm_provider"`
	HasPreferences   bool `json:"has_preferences"`
	HasFamilyMembers bool `json:"has_family_members"`
}

func (s *Server) handleGetSetupStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	hasLLM := false
	if hs, err := store.GetHouseholdSettings(ctx, s.db.Pool); err == nil && hs.LLMProvider != "" {
		hasLLM = true
	}

	var prefCount int
	_ = s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM cooking_principles`).Scan(&prefCount)

	var familyCount int
	_ = s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM family_members`).Scan(&familyCount)

	writeJSON(w, http.StatusOK, setupStatusDTO{
		SetupComplete:    hasLLM,
		HasLLMProvider:   hasLLM,
		HasPreferences:   prefCount > 0,
		HasFamilyMembers: familyCount > 0,
	})
}

func (s *Server) handleSeedPreferences(w http.ResponseWriter, r *http.Request) {
	count, err := seed.Preferences(r.Context(), s.db.Pool)
	if err != nil {
		s.internalError(w, r, "seed preferences", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"seeded": count})
}
