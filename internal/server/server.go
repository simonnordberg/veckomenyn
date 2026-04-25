package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/simonnordberg/veckomenyn/internal/agent"
	"github.com/simonnordberg/veckomenyn/internal/backup"
	"github.com/simonnordberg/veckomenyn/internal/providers"
	"github.com/simonnordberg/veckomenyn/internal/store"
	"github.com/simonnordberg/veckomenyn/internal/updates"
	"github.com/simonnordberg/veckomenyn/web"
)

type Config struct {
	Addr           string
	Build          BuildInfo
	Snapshotter    *backup.Snapshotter
	BackupConfig   BackupConfigStore
	Updates        *updates.Checker
	UpdateTrigger  *updates.Trigger
	UpdateConfig   UpdateConfigStore
}

// UpdateConfigStore is the minimal surface server handlers need for the
// auto-update toggle. Decoupled from internal/store so server tests don't
// need a DB.
type UpdateConfigStore interface {
	UpdateConfig(ctx context.Context) (store.UpdateConfig, error)
	SetAutoUpdate(ctx context.Context, enabled bool) (store.UpdateConfig, error)
}

// BackupConfigStore is the minimal surface server handlers need for the
// nightly-backup toggle UI. Decoupled from internal/store so server tests
// don't need a DB.
type BackupConfigStore interface {
	BackupConfig(ctx context.Context) (backup.Config, error)
	UpdateBackupConfig(ctx context.Context, nightlyEnabled *bool, nightlyKeep *int) (backup.Config, error)
}

// BuildInfo is build metadata stamped into the binary via -ldflags. Surfaced
// at /api/version so the UI can show what's running and check for updates.
type BuildInfo struct {
	Version string
	Commit  string
	BuiltAt string
}

type Server struct {
	cfg           Config
	db            *store.DB
	log           *slog.Logger
	agent         *agent.Agent
	providers     *providers.Store
	snapshotter   *backup.Snapshotter
	backupConfig  BackupConfigStore
	updates       *updates.Checker
	updateTrigger *updates.Trigger
	updateConfig  UpdateConfigStore
	router        *chi.Mux
	http          *http.Server
}

func New(cfg Config, db *store.DB, ag *agent.Agent, providers *providers.Store, log *slog.Logger) *Server {
	s := &Server{
		cfg:           cfg,
		db:            db,
		agent:         ag,
		providers:     providers,
		snapshotter:   cfg.Snapshotter,
		backupConfig:  cfg.BackupConfig,
		updates:       cfg.Updates,
		updateTrigger: cfg.UpdateTrigger,
		updateConfig:  cfg.UpdateConfig,
		log:           log,
		router:        chi.NewRouter(),
	}
	s.routes()
	s.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Use(noSniff)
	// Compression must not be applied to SSE, it breaks the event stream.
	// Chi's compress middleware auto-skips text/event-stream, but leaving it
	// off the /api/chat route entirely is safer.
	s.router.Use(middleware.Compress(5))
	// Cap body size globally. 1 MiB is already an order of magnitude larger
	// than anything the UI legitimately sends (long chat messages, notes MD).
	s.router.Use(middleware.RequestSize(1 << 20))

	// Rate limit the expensive endpoints per client IP. At family/LAN scale
	// anything higher than this is almost certainly a bug or abuse; the
	// Anthropic and Willys API costs are what we're protecting against.
	chatLimiter := httprate.LimitByIP(20, time.Minute)

	s.router.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/version", s.handleVersion)
		r.Get("/updates", s.handleGetUpdates)
		r.Post("/updates/apply", s.handleApplyUpdate)
		r.Get("/update-config", s.handleGetUpdateConfig)
		r.Patch("/update-config", s.handlePatchUpdateConfig)
		r.Get("/setup-status", s.handleGetSetupStatus)
		r.Post("/preferences/seed", s.handleSeedPreferences)
		r.With(chatLimiter).Post("/chat", s.handleChat)
		r.Get("/conversations", s.handleListConversations)
		r.Get("/conversations/{id}", s.handleGetConversation)
		r.Get("/weeks", s.handleListWeeks)
		r.Post("/weeks", s.handleCreateWeek)
		r.Get("/weeks/current", s.handleCurrentWeek)
		r.Get("/weeks/{iso}", s.handleGetWeek)
		r.Get("/weeks/id/{id}", s.handleGetWeekByID)
		r.Patch("/weeks/id/{id}", s.handlePatchWeek)
		r.Delete("/weeks/id/{id}", s.handleDeleteWeek)
		r.Post("/weeks/id/{id}/clone", s.handleCloneWeek)
		r.Get("/weeks/id/{id}/conversation", s.handleGetWeekConversation)
		r.Delete("/weeks/id/{id}/conversations", s.handleDeleteWeekConversations)
		r.Put("/weeks/id/{id}/retrospective", s.handlePutWeekRetrospective)
		r.Put("/dinners/id/{id}/rating", s.handlePutDinnerRating)
		r.Delete("/dinners/id/{id}/rating", s.handleDeleteDinnerRating)
		r.Get("/settings", s.handleGetSettings)
		r.Patch("/settings", s.handlePatchSettings)
		r.Get("/providers", s.handleListProviders)
		r.Patch("/providers/{kind}", s.handlePatchProvider)
		r.Get("/preferences", s.handleListPreferences)
		r.Put("/preferences/{category}", s.handlePutPreference)
		r.Delete("/preferences/{category}", s.handleDeletePreference)
		r.Get("/usage/summary", s.handleGetUsageSummary)
		r.Get("/backups", s.handleListBackups)
		r.Post("/backups", s.handleCreateBackup)
		r.Get("/backups/{filename}/download", s.handleDownloadBackup)
		r.Delete("/backups/{filename}", s.handleDeleteBackup)
		r.Get("/backup-config", s.handleGetBackupConfig)
		r.Patch("/backup-config", s.handlePatchBackupConfig)
	})

	s.router.NotFound(s.handleStatic)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "veckomenyn",
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":  s.cfg.Build.Version,
		"commit":   s.cfg.Build.Commit,
		"built_at": s.cfg.Build.BuiltAt,
	})
}

func (s *Server) handleGetUpdates(w http.ResponseWriter, r *http.Request) {
	canApply := s.updateTrigger != nil && s.updateTrigger.Configured()

	autoEnabled := false
	if s.updateConfig != nil {
		if cfg, err := s.updateConfig.UpdateConfig(r.Context()); err == nil {
			autoEnabled = cfg.AutoUpdateEnabled
		}
	}

	if s.updates == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"current":      s.cfg.Build.Version,
			"latest":       "",
			"has_update":   false,
			"url":          "",
			"can_apply":    canApply,
			"auto_enabled": autoEnabled,
		})
		return
	}
	status, err := s.updates.Status(r.Context())
	if err != nil {
		s.log.Warn("update check failed", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"current":      status.Current,
		"latest":       status.Latest,
		"has_update":   status.HasUpdate,
		"url":          status.URL,
		"can_apply":    canApply,
		"auto_enabled": autoEnabled,
	})
}

func (s *Server) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	if s.updateTrigger == nil || !s.updateTrigger.Configured() {
		http.Error(w, "update trigger not configured", http.StatusServiceUnavailable)
		return
	}
	if err := s.updateTrigger.Fire(r.Context()); err != nil {
		s.internalError(w, r, "fire update trigger", err)
		return
	}
	// 202 because the trigger returned but the actual recreate happens
	// asynchronously and our process is about to die. Caller polls
	// /api/version until the new version reports back.
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleGetUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if s.updateConfig == nil {
		http.Error(w, "update config unavailable", http.StatusServiceUnavailable)
		return
	}
	cfg, err := s.updateConfig.UpdateConfig(r.Context())
	if err != nil {
		s.internalError(w, r, "read update config", err)
		return
	}
	canApply := s.updateTrigger != nil && s.updateTrigger.Configured()
	writeJSON(w, http.StatusOK, map[string]any{
		"auto_update_enabled": cfg.AutoUpdateEnabled,
		"can_apply":           canApply,
	})
}

func (s *Server) handlePatchUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if s.updateConfig == nil {
		http.Error(w, "update config unavailable", http.StatusServiceUnavailable)
		return
	}
	var p struct {
		AutoUpdateEnabled *bool `json:"auto_update_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if p.AutoUpdateEnabled == nil {
		http.Error(w, "auto_update_enabled required", http.StatusBadRequest)
		return
	}
	if *p.AutoUpdateEnabled && (s.updateTrigger == nil || !s.updateTrigger.Configured()) {
		http.Error(w, "cannot enable auto-update: trigger not configured", http.StatusBadRequest)
		return
	}
	cfg, err := s.updateConfig.SetAutoUpdate(r.Context(), *p.AutoUpdateEnabled)
	if err != nil {
		s.internalError(w, r, "set auto_update_enabled", err)
		return
	}
	canApply := s.updateTrigger != nil && s.updateTrigger.Configured()
	writeJSON(w, http.StatusOK, map[string]any{
		"auto_update_enabled": cfg.AutoUpdateEnabled,
		"can_apply":           canApply,
	})
}

// handleStatic serves the embedded SPA. If the frontend hasn't been built
// (only .gitkeep is present), returns a placeholder page. For unknown paths
// that aren't real assets, falls back to index.html so client-side routes
// (e.g. /weeks/2026-W17) survive a reload or direct visit.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	dist, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		http.Error(w, "embed error", http.StatusInternalServerError)
		return
	}

	if _, err := fs.Stat(dist, "index.html"); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(placeholderHTML))
		return
	}

	// Strip the leading slash to match the embedded FS layout, then check
	// whether the requested file exists. If not (or if it's a directory),
	// serve index.html so the client-side router owns the URL.
	clean := strings.TrimPrefix(r.URL.Path, "/")
	if clean == "" {
		clean = "index.html"
	}
	if info, err := fs.Stat(dist, clean); err != nil || info.IsDir() {
		data, readErr := fs.ReadFile(dist, "index.html")
		if readErr != nil {
			http.Error(w, "index not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
		return
	}

	http.FileServer(http.FS(dist)).ServeHTTP(w, r)
}

func (s *Server) Start() error {
	s.log.Info("listening", "addr", s.cfg.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// noSniff tells the browser to trust our Content-Type headers rather than
// guessing from the body. Cheap defence against a future bug that serves
// HTML under an API route, or serves unexpected bytes out of the embedded
// SPA handler.
func noSniff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// internalError logs the underlying error and returns a generic 500 to the
// client. Use for unexpected server-side failures. For predictable 4xx
// conditions (bad JSON, missing fields, unknown kinds) prefer http.Error
// with a descriptive message; those are safe to return verbatim.
func (s *Server) internalError(w http.ResponseWriter, r *http.Request, what string, err error) {
	s.log.Error(what, "err", err, "path", r.URL.Path, "method", r.Method)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// parsePositiveID reads a path param and parses it as a positive int64.
// Writes a 400 and returns ok=false on anything non-numeric or <= 0.
func parsePositiveID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := chi.URLParam(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "bad "+name, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// ISO-8601 week identifier, e.g. "2025-W03". We accept weeks 01-53.
var isoWeekRE = regexp.MustCompile(`^\d{4}-W(0[1-9]|[1-4]\d|5[0-3])$`)

// Calendar date, YYYY-MM-DD. Postgres will still reject impossible dates
// (e.g. 2026-02-30); this regex only catches obviously malformed input.
var dateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

const placeholderHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Veckomenyn</title>
<style>
  :root { color-scheme: light dark; }
  html, body { margin: 0; height: 100%; font: 16px/1.5 -apple-system, system-ui, sans-serif; }
  body { display: grid; place-items: center; }
  main { max-width: 40rem; padding: 2rem; }
  code { background: rgba(128,128,128,0.2); padding: 0.1em 0.4em; border-radius: 4px; }
  h1 { margin-top: 0; }
</style>
</head>
<body>
<main>
  <h1>Veckomenyn</h1>
  <p>Server is running. Frontend is not built yet.</p>
  <p>To build the frontend:</p>
  <pre>cd web &amp;&amp; pnpm install &amp;&amp; pnpm build</pre>
  <p>Then rebuild the server binary.</p>
  <p>API health: <a href="/api/health">/api/health</a></p>
</main>
</body>
</html>
`
