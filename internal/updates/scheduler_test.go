package updates

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type fakeConfig struct {
	enabled bool
	err     error
}

func (f fakeConfig) AutoUpdateEnabled(ctx context.Context) (bool, error) {
	return f.enabled, f.err
}

// stubChecker exposes a checker pre-loaded with a Status so tick() returns
// it without making a real GitHub request.
func stubChecker(latest, current string) *Checker {
	c := New("owner/repo", current)
	c.cached = &Status{Current: current, Latest: latest, HasUpdate: cmp(latest, current) > 0}
	c.fetchedAt = time.Now()
	return c
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeTriggerSrv counts how many times Fire would be called by spinning
// up an httptest server that increments on POST.
func fakeTriggerSrv(t *testing.T, count *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAutoUpdater_DisabledDoesNothing(t *testing.T) {
	var hits atomic.Int32
	srv := fakeTriggerSrv(t, &hits)

	au := NewAutoUpdater(
		stubChecker("0.4.0", "0.3.0"),
		NewTrigger(srv.URL, ""),
		fakeConfig{enabled: false},
		newDiscardLogger(),
	)
	au.tick(t.Context())

	if got := hits.Load(); got != 0 {
		t.Errorf("trigger fired %d times, want 0 (auto disabled)", got)
	}
}

func TestAutoUpdater_NoTriggerConfigured(t *testing.T) {
	au := NewAutoUpdater(
		stubChecker("0.4.0", "0.3.0"),
		NewTrigger("", ""),
		fakeConfig{enabled: true},
		newDiscardLogger(),
	)
	// Should be a no-op (no Configured trigger).
	au.tick(t.Context())
}

func TestAutoUpdater_FiresWhenEnabledAndUpdateAvailable(t *testing.T) {
	var hits atomic.Int32
	srv := fakeTriggerSrv(t, &hits)

	au := NewAutoUpdater(
		stubChecker("0.4.0", "0.3.0"),
		NewTrigger(srv.URL, ""),
		fakeConfig{enabled: true},
		newDiscardLogger(),
	)
	au.tick(t.Context())

	if got := hits.Load(); got != 1 {
		t.Errorf("trigger fired %d times, want 1", got)
	}
}

func TestAutoUpdater_NoUpdateAvailable(t *testing.T) {
	var hits atomic.Int32
	srv := fakeTriggerSrv(t, &hits)

	au := NewAutoUpdater(
		stubChecker("0.3.0", "0.3.0"),
		NewTrigger(srv.URL, ""),
		fakeConfig{enabled: true},
		newDiscardLogger(),
	)
	au.tick(t.Context())

	if got := hits.Load(); got != 0 {
		t.Errorf("trigger fired %d times, want 0 (already current)", got)
	}
}

func TestAutoUpdater_DebouncesWithin23Hours(t *testing.T) {
	var hits atomic.Int32
	srv := fakeTriggerSrv(t, &hits)

	au := NewAutoUpdater(
		stubChecker("0.4.0", "0.3.0"),
		NewTrigger(srv.URL, ""),
		fakeConfig{enabled: true},
		newDiscardLogger(),
	)
	au.tick(t.Context())
	// Even if checker still says has_update, the debounce should suppress.
	au.tick(t.Context())

	if got := hits.Load(); got != 1 {
		t.Errorf("trigger fired %d times across two ticks within 23h, want 1", got)
	}
}
