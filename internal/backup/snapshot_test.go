package backup

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSanitizeLabel(t *testing.T) {
	cases := map[string]string{
		"v0.1.0":        "v0.1.0",
		"Pre Migration": "pre-migration",
		"":              "untagged",
		"../etc/passwd": "..-etc-passwd",
		"Hello World!":  "hello-world-",
		"WeEk-2026-W17": "week-2026-w17",
		"upper.UPPER":   "upper.upper",
	}
	for in, want := range cases {
		if got := sanitizeLabel(in); got != want {
			t.Errorf("sanitizeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseFilename(t *testing.T) {
	cases := []struct {
		name       string
		wantReason Reason
		wantLabel  string
	}{
		{"20260425T100000Z_pre-migration_v0.2.0.dump", "pre-migration", "v0.2.0"},
		{"20260425T100000Z_manual_untagged.dump", "manual", "untagged"},
		{"20260425T100000Z_nightly_2026-04-25.dump", "nightly", "2026-04-25"},
		{"20260425T100000Z_pre-migration_label.with.dots.dump", "pre-migration", "label.with.dots"},
		{"weird.dump", "", ""},
		{"20260425T100000Z_manual.dump", "manual", ""},
	}
	for _, c := range cases {
		gotR, gotL := parseFilename(c.name)
		if gotR != c.wantReason || gotL != c.wantLabel {
			t.Errorf("parseFilename(%q) = (%q, %q), want (%q, %q)",
				c.name, gotR, gotL, c.wantReason, c.wantLabel)
		}
	}
}

func TestListAndPrune(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s, err := Open(dir, logger)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Create fake dumps with deterministic mtimes; newest first when listed.
	type fake struct {
		name  string
		mtime time.Time
	}
	now := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	fakes := []fake{
		{"20260425T100000Z_pre-migration_v0.5.0.dump", now},
		{"20260424T100000Z_pre-migration_v0.4.0.dump", now.Add(-24 * time.Hour)},
		{"20260423T100000Z_pre-migration_v0.3.0.dump", now.Add(-48 * time.Hour)},
		{"20260422T100000Z_pre-migration_v0.2.0.dump", now.Add(-72 * time.Hour)},
		{"20260425T090000Z_manual_alpha.dump", now.Add(-1 * time.Hour)},
		{"20260420T100000Z_nightly_2026-04-20.dump", now.Add(-120 * time.Hour)},
		{"not-a-dump.txt", now}, // ignored
	}
	for _, f := range fakes {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
			t.Fatalf("write fake: %v", err)
		}
		if err := os.Chtimes(path, f.mtime, f.mtime); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 6 {
		t.Fatalf("want 6 snapshots, got %d", len(all))
	}
	if all[0].Filename != "20260425T100000Z_pre-migration_v0.5.0.dump" {
		t.Errorf("want newest first, got %q", all[0].Filename)
	}

	// Keep 2 pre-migration → drop the two oldest pre-migration, leave manual
	// and nightly intact.
	removed, err := s.Prune(ReasonPreMigration, 2)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("want 2 removed, got %d (%v)", len(removed), removed)
	}

	after, err := s.List()
	if err != nil {
		t.Fatalf("List after prune: %v", err)
	}
	if len(after) != 4 {
		t.Fatalf("want 4 remaining, got %d", len(after))
	}
	for _, snap := range after {
		// Confirm we kept the two newest pre-migration and untouched others.
		if snap.Filename == "20260423T100000Z_pre-migration_v0.3.0.dump" ||
			snap.Filename == "20260422T100000Z_pre-migration_v0.2.0.dump" {
			t.Errorf("expected %q to be pruned", snap.Filename)
		}
	}
}

func TestPruneKeepZeroErrors(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s.Prune(ReasonPreMigration, 0); err == nil {
		t.Error("Prune with keep=0 should error")
	}
}

func TestSnapshotReadOnlyErrors(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s.Snapshot(t.Context(), ReasonManual, "x"); err == nil {
		t.Error("Snapshot on read-only Snapshotter should error")
	}
}
