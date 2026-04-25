// Package backup takes logical Postgres dumps using pg_dump's custom format.
// Restores with stock pg_restore. Snapshots are written to a directory the
// caller mounts as a Docker volume so they survive container replacement.
package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Reason is the trigger for a snapshot. It's encoded in the filename so
// retention can be applied per-reason and the UI can group them.
type Reason string

const (
	ReasonPreMigration Reason = "pre-migration"
	ReasonManual       Reason = "manual"
	ReasonNightly      Reason = "nightly"
)

type Snapshotter struct {
	dsn    string
	dir    string
	log    *slog.Logger
	pgDump string
}

type Snapshot struct {
	Path      string    `json:"-"`
	Filename  string    `json:"filename"`
	Reason    Reason    `json:"reason"`
	Label     string    `json:"label"`
	Size      int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// Open prepares the snapshot directory for reads (List, Prune). It does
// not require pg_dump to be available.
func Open(dir string, log *slog.Logger) (*Snapshotter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir %q: %w", dir, err)
	}
	return &Snapshotter{dir: dir, log: log}, nil
}

// New prepares the snapshot directory and resolves pg_dump for taking new
// snapshots. Returns an error if pg_dump is not on PATH.
func New(dsn, dir string, log *slog.Logger) (*Snapshotter, error) {
	s, err := Open(dir, log)
	if err != nil {
		return nil, err
	}
	pgDump, err := exec.LookPath("pg_dump")
	if err != nil {
		return nil, fmt.Errorf("pg_dump not on PATH: %w", err)
	}
	s.dsn = dsn
	s.pgDump = pgDump
	return s, nil
}

// Snapshot runs pg_dump in custom format. Output: {ts}_{reason}_{label}.dump,
// restorable with `pg_restore --clean --if-exists --no-owner --no-privileges
// -d $DSN file.dump`. Returns metadata about the file.
func (s *Snapshotter) Snapshot(ctx context.Context, reason Reason, label string) (Snapshot, error) {
	if s.pgDump == "" {
		return Snapshot{}, fmt.Errorf("snapshotter opened read-only; use New() to take backups")
	}

	safeLabel := sanitizeLabel(label)
	filename := fmt.Sprintf("%s_%s_%s.dump",
		time.Now().UTC().Format("20060102T150405Z"),
		reason,
		safeLabel,
	)
	path := filepath.Join(s.dir, filename)

	cmd := exec.CommandContext(ctx, s.pgDump,
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"--file="+path,
		"--dbname="+s.dsn,
	)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		_ = os.Remove(path)
		return Snapshot{}, fmt.Errorf("pg_dump: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	info, err := os.Stat(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("stat dump: %w", err)
	}

	s.log.Info("snapshot taken",
		"file", filename,
		"size_bytes", info.Size(),
		"reason", reason,
		"label", label,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return Snapshot{
		Path:      path,
		Filename:  filename,
		Reason:    reason,
		Label:     label,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
	}, nil
}

// List returns existing snapshots, newest first.
func (s *Snapshotter) List() ([]Snapshot, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	var snaps []Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".dump") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		reason, label := parseFilename(e.Name())
		snaps = append(snaps, Snapshot{
			Path:      filepath.Join(s.dir, e.Name()),
			Filename:  e.Name(),
			Reason:    reason,
			Label:     label,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].CreatedAt.After(snaps[j].CreatedAt)
	})
	return snaps, nil
}

// Prune removes snapshots beyond the keep count for the given reason. Other
// reasons are left untouched. Returns the filenames that were removed.
func (s *Snapshotter) Prune(reason Reason, keep int) ([]string, error) {
	if keep <= 0 {
		return nil, fmt.Errorf("keep must be > 0, got %d", keep)
	}
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	var ofReason []Snapshot
	for _, snap := range all {
		if snap.Reason == reason {
			ofReason = append(ofReason, snap)
		}
	}
	if len(ofReason) <= keep {
		return nil, nil
	}
	var removed []string
	for _, snap := range ofReason[keep:] {
		if err := os.Remove(snap.Path); err != nil {
			return removed, fmt.Errorf("remove %q: %w", snap.Filename, err)
		}
		removed = append(removed, snap.Filename)
	}
	return removed, nil
}

// Dir is the directory snapshots are written to.
func (s *Snapshotter) Dir() string { return s.dir }

func sanitizeLabel(in string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(in) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	if out == "" {
		return "untagged"
	}
	return out
}

func parseFilename(name string) (Reason, string) {
	base := strings.TrimSuffix(name, ".dump")
	parts := strings.SplitN(base, "_", 3)
	switch len(parts) {
	case 0, 1:
		return "", ""
	case 2:
		return Reason(parts[1]), ""
	default:
		return Reason(parts[1]), parts[2]
	}
}
