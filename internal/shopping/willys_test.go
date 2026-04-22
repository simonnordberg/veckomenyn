package shopping

import "testing"

func TestParseKronor(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"", 0},
		{"abc", 0},
		{"12,50", 12.5},
		{"1.299,00", 1299},
		{"1 299,00", 1299},
		{"1 299,00 kr", 1299}, // non-breaking space often shows up in SE prices
		{"47,95 kr", 47.95},
		{"0,00", 0},
		{"100", 100},
	}
	for _, c := range cases {
		got := parseKronor(c.in)
		if got != c.want {
			t.Errorf("parseKronor(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "x", "y"); got != "x" {
		t.Fatalf("expected x, got %q", got)
	}
	if got := firstNonEmpty(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
