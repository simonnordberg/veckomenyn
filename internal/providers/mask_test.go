package providers

import (
	"strings"
	"testing"
)

func TestNewGeneratesSentinel(t *testing.T) {
	s1, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s2, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if s1.Sentinel() == "" {
		t.Fatal("sentinel must be non-empty")
	}
	if !strings.HasPrefix(s1.Sentinel(), "redacted:") {
		t.Fatalf("sentinel %q missing redacted: prefix", s1.Sentinel())
	}
	if s1.Sentinel() == s2.Sentinel() {
		t.Fatal("two fresh stores produced identical sentinels — rand.Read not consumed")
	}
}

func TestMaskReplacesPasswordFields(t *testing.T) {
	s, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := Provider{
		Kind: KindWillys,
		Config: map[string]any{
			"username": "198001011234",
			"password": "hunter2",
		},
	}
	m := s.Mask(p)
	if m.Config["username"] != "198001011234" {
		t.Fatalf("username should not be masked, got %v", m.Config["username"])
	}
	if m.Config["password"] != s.Sentinel() {
		t.Fatalf("password should equal sentinel, got %v", m.Config["password"])
	}
	// Original must be untouched.
	if p.Config["password"] != "hunter2" {
		t.Fatal("Mask mutated its input")
	}
}

func TestMaskLeavesEmptyPasswordFieldEmpty(t *testing.T) {
	s, _ := New(nil, nil)
	p := Provider{Kind: KindAnthropic, Config: map[string]any{"api_key": ""}}
	m := s.Mask(p)
	if m.Config["api_key"] != "" {
		t.Fatalf("empty secret should stay empty, got %v", m.Config["api_key"])
	}
}

func TestMaskFallsBackToKeyNameHeuristics(t *testing.T) {
	// Unknown kind → Mask should still redact common secret-named keys so
	// a misconfigured provider can't accidentally leak.
	s, _ := New(nil, nil)
	p := Provider{
		Kind: "unknown-kind",
		Config: map[string]any{
			"api_key": "abc",
			"token":   "xyz",
			"other":   "visible",
		},
	}
	m := s.Mask(p)
	if m.Config["api_key"] != s.Sentinel() {
		t.Fatalf("api_key not masked: %v", m.Config["api_key"])
	}
	if m.Config["token"] != s.Sentinel() {
		t.Fatalf("token not masked: %v", m.Config["token"])
	}
	if m.Config["other"] != "visible" {
		t.Fatalf("non-secret key masked: %v", m.Config["other"])
	}
}
