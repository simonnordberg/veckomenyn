package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/simonnordberg/veckomenyn/internal/providers"
)

// providersEnvelope wraps the list response with kind metadata so the UI can
// render forms without hard-coding field lists. Sentinel is the random
// per-process string returned in place of set secret fields; the UI echoes
// it back on PATCH to mean "leave this secret alone".
type providersEnvelope struct {
	Known     []providers.KindInfo `json:"known"`
	Providers []providers.Provider `json:"providers"`
	Sentinel  string               `json:"sentinel"`
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	list, err := s.providers.List(r.Context())
	if err != nil {
		s.internalError(w, r, "request", err)
		return
	}
	// Ensure every Known kind shows up even if the DB has no row yet.
	have := map[providers.Kind]bool{}
	for _, p := range list {
		have[p.Kind] = true
	}
	for _, info := range providers.Known {
		if !have[info.Kind] {
			list = append(list, providers.Provider{
				Kind:    info.Kind,
				Enabled: false,
				Config:  map[string]any{},
			})
		}
	}
	masked := make([]providers.Provider, len(list))
	for i, p := range list {
		masked[i] = s.providers.Mask(p)
	}
	writeJSON(w, http.StatusOK, providersEnvelope{
		Known:     providers.Known,
		Providers: masked,
		Sentinel:  s.providers.Sentinel(),
	})
}

type providerPatch struct {
	Enabled *bool          `json:"enabled"`
	Config  map[string]any `json:"config"`
}

func (s *Server) handlePatchProvider(w http.ResponseWriter, r *http.Request) {
	kind := providers.Kind(chi.URLParam(r, "kind"))
	if _, ok := providers.KindInfoFor(kind); !ok {
		http.Error(w, "unknown provider kind", http.StatusBadRequest)
		return
	}
	var p providerPatch
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	out, err := s.providers.Upsert(r.Context(), kind, providers.UpsertPatch{
		Enabled: p.Enabled,
		Config:  p.Config,
	})
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		s.internalError(w, r, "request", err)
		return
	}
	writeJSON(w, http.StatusOK, s.providers.Mask(*out))
}
