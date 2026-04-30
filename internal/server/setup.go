package server

import (
	"net/http"

	"github.com/simonnordberg/veckomenyn/internal/seed"
)

// setupStatusDTO drives the first-run wizard. setup_complete flips to true
// the moment any LLM provider is configured. Households can be added later
// via the regular UI; we don't gate on them.
type setupStatusDTO struct {
	SetupComplete    bool `json:"setup_complete"`
	HasLLMProvider   bool `json:"has_llm_provider"`
	HasPreferences   bool `json:"has_preferences"`
	HasFamilyMembers bool `json:"has_family_members"`
}

func (s *Server) handleGetSetupStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hasAnthropic := s.providers.AnthropicAPIKey(ctx) != ""
	_, hasOpenAI := s.providers.OpenAICompatConfig(ctx)
	hasLLM := hasAnthropic || hasOpenAI

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
