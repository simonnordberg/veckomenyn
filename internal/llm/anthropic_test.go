package llm

import "testing"

func TestNewAnthropic_EmptyKey(t *testing.T) {
	_, err := NewAnthropic("")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestNewAnthropic_ValidKey(t *testing.T) {
	p, err := NewAnthropic("sk-test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("provider should not be nil")
	}
}

func TestAnthropicProvider_ImplementsProvider(t *testing.T) {
	p, _ := NewAnthropic("sk-test-key")
	var _ Provider = p
}
