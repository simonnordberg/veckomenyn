package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestIsoWeekRegex(t *testing.T) {
	good := []string{"2024-W01", "2025-W53", "1999-W49"}
	for _, s := range good {
		if !isoWeekRE.MatchString(s) {
			t.Errorf("expected match for %q", s)
		}
	}
	bad := []string{
		"",
		"2024",
		"2024-1",
		"2024-W0",
		"2024-W00",
		"2024-W54",
		"2024-w01",
		"2024-W1",
		"abcd-W01",
		"2024/W01",
		"2024-W01 ",
		" 2024-W01",
	}
	for _, s := range bad {
		if isoWeekRE.MatchString(s) {
			t.Errorf("expected no match for %q", s)
		}
	}
}

func TestParsePositiveID(t *testing.T) {
	runCase := func(raw string, wantOK bool, wantID int64) {
		t.Helper()
		r := chi.NewRouter()
		var gotID int64
		var gotOK bool
		r.Get("/x/{id}", func(w http.ResponseWriter, req *http.Request) {
			gotID, gotOK = parsePositiveID(w, req, "id")
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/x/"+raw, nil)
		r.ServeHTTP(w, req)
		if gotOK != wantOK {
			t.Fatalf("raw=%q ok=%v want %v (body=%q)", raw, gotOK, wantOK, w.Body.String())
		}
		if wantOK && gotID != wantID {
			t.Fatalf("raw=%q id=%d want %d", raw, gotID, wantID)
		}
		if !wantOK && w.Code != http.StatusBadRequest {
			t.Fatalf("raw=%q expected 400, got %d", raw, w.Code)
		}
	}
	runCase("1", true, 1)
	runCase("42", true, 42)
	runCase("0", false, 0)
	runCase("-1", false, 0)
	runCase("abc", false, 0)
	runCase("1.5", false, 0)
	runCase("99999999999999999999", false, 0) // overflow
}

func TestNoSniffMiddleware(t *testing.T) {
	h := noSniff(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options=nosniff, got %q", got)
	}
}
