package updates

import "testing"

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
