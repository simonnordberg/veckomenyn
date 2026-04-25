// Package updates polls GitHub releases for newer versions of veckomenyn
// and surfaces an upgrade hint to the UI. Cached in-memory with 1h TTL so
// every install is at most one upstream call per hour and serves stale on
// transient failure rather than spamming GitHub or breaking the banner.
package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Status struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	HasUpdate bool   `json:"has_update"`
	URL       string `json:"url"`
}

type Checker struct {
	repo       string
	current    string
	httpClient *http.Client
	ttl        time.Duration

	mu        sync.Mutex
	cached    *Status
	fetchedAt time.Time
}

// New configures a Checker for the given GitHub repo (e.g.
// "simonnordberg/veckomenyn") and current version string. current is the
// build's embedded version; non-semver values (e.g. "dev") never report an
// update available.
func New(repo, current string) *Checker {
	return &Checker{
		repo:       repo,
		current:    current,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		ttl:        time.Hour,
	}
}

// Status returns the cached upgrade status, refreshing if past TTL. Errors
// are absorbed and the previous value is served, so a 30 second GitHub
// outage shouldn't break the banner.
func (c *Checker) Status(ctx context.Context) (Status, error) {
	c.mu.Lock()
	if c.cached != nil && time.Since(c.fetchedAt) < c.ttl {
		s := *c.cached
		c.mu.Unlock()
		return s, nil
	}
	c.mu.Unlock()

	latest, url, err := c.fetchLatest(ctx)
	if err != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.cached != nil {
			return *c.cached, nil
		}
		// First attempt failed; return current-only state without claiming
		// an update is available.
		return Status{Current: c.current}, err
	}

	status := Status{
		Current:   c.current,
		Latest:    latest,
		HasUpdate: cmp(latest, c.current) > 0,
		URL:       url,
	}
	c.mu.Lock()
	c.cached = &status
	c.fetchedAt = time.Now()
	c.mu.Unlock()
	return status, nil
}

func (c *Checker) fetchLatest(ctx context.Context) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.github.com/repos/"+c.repo+"/releases/latest", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "veckomenyn-update-check")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github releases: status %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", fmt.Errorf("decode: %w", err)
	}
	return strings.TrimPrefix(body.TagName, "v"), body.HTMLURL, nil
}

// cmp returns 1 if a > b, -1 if a < b, 0 if equal or unparsable. Strips a
// leading "v" and ignores pre-release suffixes; sufficient for the
// "are we behind the latest tag" question.
func cmp(a, b string) int {
	pa, oka := parseSemver(a)
	pb, okb := parseSemver(b)
	if !oka || !okb {
		return 0
	}
	for i := 0; i < 3; i++ {
		switch {
		case pa[i] > pb[i]:
			return 1
		case pa[i] < pb[i]:
			return -1
		}
	}
	return 0
}

func parseSemver(s string) ([3]int, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
