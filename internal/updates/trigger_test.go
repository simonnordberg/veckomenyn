package updates

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrigger_Configured(t *testing.T) {
	if (&Trigger{}).Configured() {
		t.Error("empty trigger should report Configured() = false")
	}
	if !NewTrigger("http://x/y", "tok").Configured() {
		t.Error("non-empty url should report Configured() = true")
	}
}

func TestTrigger_FireNotConfigured(t *testing.T) {
	tr := NewTrigger("", "")
	if err := tr.Fire(t.Context()); err == nil {
		t.Error("Fire on unconfigured trigger should error")
	}
}

func TestTrigger_FireSends(t *testing.T) {
	var (
		gotPath   string
		gotMethod string
		gotAuth   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewTrigger(srv.URL+"/v1/update", "secret-token")
	if err := tr.Fire(t.Context()); err != nil {
		t.Fatalf("Fire: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/update" {
		t.Errorf("path = %q, want /v1/update", gotPath)
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("auth = %q, want Bearer secret-token", gotAuth)
	}
}

func TestTrigger_FireNoToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewTrigger(srv.URL+"/v1/update", "")
	if err := tr.Fire(t.Context()); err != nil {
		t.Fatalf("Fire: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("auth = %q, want empty (no token configured)", gotAuth)
	}
}

func TestTrigger_FireServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	tr := NewTrigger(srv.URL+"/v1/update", "tok")
	err := tr.Fire(t.Context())
	if err == nil {
		t.Fatal("Fire should error on 403")
	}
	if got := err.Error(); got == "" {
		t.Errorf("error message empty, want context")
	}
}
