package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/simonnordberg/veckomenyn/internal/llm"
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

func (s *Server) handleTestProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Find the currently enabled LLM provider from the DB.
	llmKinds := []providers.Kind{providers.KindAnthropic, providers.KindOpenAI, providers.KindOpenAICompat}
	var activeKind providers.Kind
	var config map[string]any
	for _, kind := range llmKinds {
		p, err := s.providers.Get(ctx, kind)
		if err != nil || !p.Enabled {
			continue
		}
		activeKind = kind
		config = p.Config
		break
	}
	if activeKind == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": "No LLM provider enabled"})
		return
	}

	provider, model, err := buildProvider(activeKind, config)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var reply string
	result, err := provider.RunStream(testCtx, llm.RunParams{
		Model:     model,
		MaxTokens: 200,
		System:    []llm.SystemBlock{{Text: "You are a meal-planning assistant. The user is testing that you are reachable. Introduce yourself in one short sentence in the user's language. Be warm and mention food."}},
		Messages:  []llm.Message{llm.NewUserMessage(llm.TextBlock("Hej! Fungerar du?"))},
	}, func(ev llm.StreamEvent) {
		if ev.Kind == llm.EventTextDelta {
			reply += ev.Text
		}
	})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"model": model,
		"reply": reply,
		"usage": map[string]any{
			"input_tokens":  result.Usage.InputTokens,
			"output_tokens": result.Usage.OutputTokens,
		},
	})
}

func buildProvider(kind providers.Kind, config map[string]any) (llm.Provider, string, error) {
	str := func(key string) string {
		v, _ := config[key].(string)
		return v
	}
	switch kind {
	case providers.KindAnthropic:
		key := str("api_key")
		if key == "" {
			return nil, "", errors.New("API key required")
		}
		model := str("model")
		if model == "" {
			model = providers.DefaultAnthropicModel
		}
		p, err := llm.NewAnthropic(key)
		return p, model, err

	case providers.KindOpenAI:
		key := str("api_key")
		if key == "" {
			return nil, "", errors.New("API key required")
		}
		model := str("model")
		if model == "" {
			model = providers.DefaultOpenAIModel
		}
		p, err := llm.NewOpenAI("https://api.openai.com/v1", model, key)
		return p, model, err

	case providers.KindOpenAICompat:
		baseURL := str("base_url")
		if baseURL == "" {
			return nil, "", errors.New("base URL required")
		}
		model := str("model")
		if model == "" {
			return nil, "", errors.New("model required")
		}
		p, err := llm.NewOpenAI(baseURL, model, str("api_key"))
		return p, model, err

	default:
		return nil, "", errors.New("unsupported provider kind for testing")
	}
}
