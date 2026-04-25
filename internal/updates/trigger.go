package updates

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Trigger asks an external orchestrator (typically Watchtower in trigger-only
// mode) to pull the latest image of veckomenyn and recreate the container.
// Fire-and-forget from the app's perspective: the trigger returns quickly,
// the actual restart happens out-of-band, and the app process dies mid-way
// through. The caller's response stream may not survive.
type Trigger struct {
	url        string
	token      string
	httpClient *http.Client
}

// NewTrigger configures the trigger. Empty url disables it; the result is
// non-nil but Configured() returns false so handlers can render UI without
// an offer to update. Empty token is allowed for setups where Watchtower's
// HTTP API is unauthenticated.
func NewTrigger(url, token string) *Trigger {
	return &Trigger{
		url:        url,
		token:      token,
		// Watchtower's /v1/update is synchronous: it doesn't return
		// until the pull + recreate finishes. Multi-arch pulls on a
		// slow link can easily run 30+ seconds. Generous timeout, but
		// callers should still spawn this in a goroutine so a slow
		// Watchtower doesn't block the HTTP handler.
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// Configured reports whether a trigger URL was set. Used by handlers to
// gate the "Update now" button on the frontend.
func (t *Trigger) Configured() bool { return t.url != "" }

// Fire posts to the trigger URL with the bearer token. Returns the HTTP
// status as an error string if non-2xx; otherwise nil. The caller should
// not assume the update has completed when this returns; the orchestrator
// runs the recreate asynchronously.
func (t *Trigger) Fire(ctx context.Context) error {
	if !t.Configured() {
		return fmt.Errorf("update trigger not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call trigger: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("trigger returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
