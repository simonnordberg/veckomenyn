package server

import (
	"strings"
	"testing"
)

func TestConversationTitle(t *testing.T) {
	if got := conversationTitle("  hi there  "); got != "hi there" {
		t.Errorf("got %q", got)
	}
	if got := conversationTitle("line one\nline two"); got != "line one" {
		t.Errorf("got %q", got)
	}
	// 80 chars = keep as-is, 81 chars = truncate with ellipsis.
	eighty := strings.Repeat("a", 80)
	if got := conversationTitle(eighty); got != eighty {
		t.Errorf("80-char title was mangled: %q", got)
	}
	eightyOne := strings.Repeat("a", 81)
	got := conversationTitle(eightyOne)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis, got %q", got)
	}
	if len([]rune(got)) != 81 { // 80 runes + 1 ellipsis rune
		t.Errorf("expected 81 runes, got %d (%q)", len([]rune(got)), got)
	}
}
