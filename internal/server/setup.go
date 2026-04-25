package server

import (
	"net/http"

	"github.com/simonnordberg/veckomenyn/internal/seed"
)

// setupStatusDTO drives the first-run wizard. setup_complete flips to true
// the moment an Anthropic key is configured. Households can be added later
// via the regular UI; we don't gate on them.
type setupStatusDTO struct {
	SetupComplete    bool `json:"setup_complete"`
	HasAnthropicKey  bool `json:"has_anthropic_key"`
	HasPreferences   bool `json:"has_preferences"`
	HasFamilyMembers bool `json:"has_family_members"`
}

func (s *Server) handleGetSetupStatus(w http.ResponseWriter, r *http.Request) {
	hasKey := s.providers.AnthropicAPIKey(r.Context()) != ""

	var prefCount int
	_ = s.db.Pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM cooking_principles`).Scan(&prefCount)

	var familyCount int
	_ = s.db.Pool.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM family_members`).Scan(&familyCount)

	writeJSON(w, http.StatusOK, setupStatusDTO{
		SetupComplete:    hasKey,
		HasAnthropicKey:  hasKey,
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
