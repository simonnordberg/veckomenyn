package willys

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestConcurrentDo exercises the RWMutex on cookies/csrfToken while many
// goroutines hit the same client. Requires -race to catch a regression in
// the locking pattern introduced when we fanned out server workers.
func TestConcurrentDo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session-id", Value: "abc"})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClientWithStore(nil)
	c.baseOverride = srv.URL

	const goroutines = 32
	const iterations = 25
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				resp, err := c.do(http.MethodGet, "/axfood/rest/csrf-token", nil)
				if err != nil {
					t.Errorf("do: %v", err)
					return
				}
				_ = resp.Body.Close()
				// Also hit the other locked paths so we cover reads + writes.
				_ = c.needsCSRF()
				c.IsLoggedIn()
			}
		}()
	}
	wg.Wait()
}

func TestClearStateResetsSession(t *testing.T) {
	c := NewClientWithStore(nil)
	c.mu.Lock()
	c.cookies["foo"] = "bar"
	c.csrfToken = "t"
	c.mu.Unlock()

	c.ClearState()

	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.cookies) != 0 {
		t.Errorf("cookies not cleared: %v", c.cookies)
	}
	if c.csrfToken != "" {
		t.Errorf("csrf not cleared: %q", c.csrfToken)
	}
}

