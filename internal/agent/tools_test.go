package agent

import "testing"

func TestEscapeILIKE(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"plain", "plain"},
		{"50%", `50\%`},
		{"pre_fix", `pre\_fix`},
		{"a%b_c", `a\%b\_c`},
		{`already\escaped`, `already\\escaped`},
		{`\%_`, `\\\%\_`},
	}
	for _, c := range cases {
		got := escapeILIKE(c.in)
		if got != c.want {
			t.Errorf("escapeILIKE(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCoalesce(t *testing.T) {
	empty := ""
	filled := "x"
	if got := coalesce(nil, "def"); got != "def" {
		t.Errorf("nil → %q, want def", got)
	}
	if got := coalesce(&empty, "def"); got != "def" {
		t.Errorf("empty → %q, want def", got)
	}
	if got := coalesce(&filled, "def"); got != "x" {
		t.Errorf("filled → %q, want x", got)
	}
}
