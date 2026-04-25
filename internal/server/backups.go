package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/simonnordberg/veckomenyn/internal/backup"
)

// snapshotterOrError writes a 503 if the backup feature is disabled and
// returns false. Most handlers can early-exit using it.
func (s *Server) snapshotterOrError(w http.ResponseWriter) (*backup.Snapshotter, bool) {
	if s.snapshotter == nil {
		http.Error(w, "backups not configured (BACKUP_DIR unset)", http.StatusServiceUnavailable)
		return nil, false
	}
	return s.snapshotter, true
}

func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	snap, ok := s.snapshotterOrError(w)
	if !ok {
		return
	}
	list, err := snap.List()
	if err != nil {
		s.internalError(w, r, "list backups", err)
		return
	}
	if list == nil {
		list = []backup.Snapshot{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": list})
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	snap, ok := s.snapshotterOrError(w)
	if !ok {
		return
	}
	saved, err := snap.Snapshot(r.Context(), backup.ReasonManual, s.cfg.Build.Version)
	if err != nil {
		s.internalError(w, r, "take snapshot", err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// resolveBackup finds a snapshot by filename in the snapshotter's listing.
// Returning matched paths from the listing avoids any path-traversal worry:
// only files we already enumerated are reachable.
func (s *Server) resolveBackup(w http.ResponseWriter, r *http.Request) (backup.Snapshot, *backup.Snapshotter, bool) {
	snap, ok := s.snapshotterOrError(w)
	if !ok {
		return backup.Snapshot{}, nil, false
	}
	filename := chi.URLParam(r, "filename")
	list, err := snap.List()
	if err != nil {
		s.internalError(w, r, "list backups", err)
		return backup.Snapshot{}, nil, false
	}
	for _, b := range list {
		if b.Filename == filename {
			return b, snap, true
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
	return backup.Snapshot{}, nil, false
}

func (s *Server) handleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	b, _, ok := s.resolveBackup(w, r)
	if !ok {
		return
	}
	f, err := os.Open(b.Path)
	if err != nil {
		s.internalError(w, r, "open backup", err)
		return
	}
	defer func() { _ = f.Close() }()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(b.Path)+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(b.Size, 10))
	if _, err := io.Copy(w, f); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		s.log.Warn("backup stream interrupted", "err", err)
	}
}

func (s *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	b, _, ok := s.resolveBackup(w, r)
	if !ok {
		return
	}
	if err := os.Remove(b.Path); err != nil {
		s.internalError(w, r, "delete backup", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type backupConfigDTO struct {
	NightlyEnabled bool `json:"nightly_enabled"`
	NightlyKeep    int  `json:"nightly_keep"`
	CanWrite       bool `json:"can_write"`
}

func (s *Server) handleGetBackupConfig(w http.ResponseWriter, r *http.Request) {
	if s.backupConfig == nil {
		http.Error(w, "backup config unavailable", http.StatusServiceUnavailable)
		return
	}
	cfg, err := s.backupConfig.BackupConfig(r.Context())
	if err != nil {
		s.internalError(w, r, "read backup config", err)
		return
	}
	writeJSON(w, http.StatusOK, backupConfigDTO{
		NightlyEnabled: cfg.NightlyEnabled,
		NightlyKeep:    cfg.NightlyKeep,
		CanWrite:       s.snapshotter != nil && s.snapshotter.CanWrite(),
	})
}

func (s *Server) handlePatchBackupConfig(w http.ResponseWriter, r *http.Request) {
	if s.backupConfig == nil {
		http.Error(w, "backup config unavailable", http.StatusServiceUnavailable)
		return
	}
	var p struct {
		NightlyEnabled *bool `json:"nightly_enabled"`
		NightlyKeep    *int  `json:"nightly_keep"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if p.NightlyEnabled != nil && *p.NightlyEnabled &&
		(s.snapshotter == nil || !s.snapshotter.CanWrite()) {
		http.Error(w, "nightly backups require pg_dump and BACKUP_DIR", http.StatusBadRequest)
		return
	}
	if p.NightlyKeep != nil && *p.NightlyKeep <= 0 {
		http.Error(w, "nightly_keep must be > 0", http.StatusBadRequest)
		return
	}
	cfg, err := s.backupConfig.UpdateBackupConfig(r.Context(), p.NightlyEnabled, p.NightlyKeep)
	if err != nil {
		s.internalError(w, r, "update backup config", err)
		return
	}
	writeJSON(w, http.StatusOK, backupConfigDTO{
		NightlyEnabled: cfg.NightlyEnabled,
		NightlyKeep:    cfg.NightlyKeep,
		CanWrite:       s.snapshotter != nil && s.snapshotter.CanWrite(),
	})
}
