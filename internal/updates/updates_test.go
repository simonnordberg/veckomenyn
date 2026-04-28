package updates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestStatusServesCachedWithinTTL(t *testing.T) {
	var calls int32
	srv := newReleaseServer(t, &calls, "v0.5.0")
	defer srv.Close()

	c := New("simonnordberg/veckomenyn", "0.4.0")
	c.httpClient = srv.Client()
	c.releasesURL = srv.URL

	if _, err := c.Status(context.Background()); err != nil {
		t.Fatalf("first Status: %v", err)
	}
	if _, err := c.Status(context.Background()); err != nil {
		t.Fatalf("second Status: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("upstream calls = %d, want 1 (cache hit on second)", got)
	}
}

func TestRefreshBypassesCache(t *testing.T) {
	var calls int32
	srv := newReleaseServer(t, &calls, "v0.5.0")
	defer srv.Close()

	c := New("simonnordberg/veckomenyn", "0.4.0")
	c.httpClient = srv.Client()
	c.releasesURL = srv.URL

	if _, err := c.Status(context.Background()); err != nil {
		t.Fatalf("Status: %v", err)
	}
	s, err := c.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("upstream calls = %d, want 2 (refresh must bypass cache)", got)
	}
	if !s.HasUpdate || s.Latest != "0.5.0" {
		t.Errorf("status = %+v, want HasUpdate with latest 0.5.0", s)
	}
	// Cache should now be primed; next Status() should not call upstream.
	if _, err := c.Status(context.Background()); err != nil {
		t.Fatalf("Status after Refresh: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("upstream calls after follow-up Status = %d, want 2 (cache hit)", got)
	}
}

func newReleaseServer(t *testing.T, calls *int32, tag string) *httptest.Server {
	t.Helper()
	body := `{"tag_name":"` + tag + `","html_url":"https://example.test/r"}`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}


func TestCmp(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.2.0", "0.1.0", 1},
		{"0.1.0", "0.2.0", -1},
		{"0.1.0", "0.1.0", 0},
		{"v0.1.0", "0.1.0", 0},
		{"0.1.10", "0.1.2", 1},
		{"1.0.0", "0.99.99", 1},
		{"0.1.0-rc1", "0.1.0", 0}, // pre-release ignored
		{"dev", "0.1.0", 0},       // unparsable
	}
	for _, c := range cases {
		if got := cmp(c.a, c.b); got != c.want {
			t.Errorf("cmp(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in   string
		want [3]int
		ok   bool
	}{
		{"0.1.0", [3]int{0, 1, 0}, true},
		{"v1.2.3", [3]int{1, 2, 3}, true},
		{"0.2.0-rc1", [3]int{0, 2, 0}, true},
		{"0.2.0+build.42", [3]int{0, 2, 0}, true},
		{"dev", [3]int{}, false},
		{"0.1", [3]int{}, false},
	}
	for _, c := range cases {
		got, ok := parseSemver(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("parseSemver(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
