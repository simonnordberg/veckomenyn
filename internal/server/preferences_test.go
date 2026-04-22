package server

import "testing"

func TestValidCategory(t *testing.T) {
	good := []string{"a", "vegetables", "meat_and_fish", "quick-dinners", "cat1"}
	for _, c := range good {
		if !validCategory(c) {
			t.Errorf("expected %q to be valid", c)
		}
	}
	bad := []string{
		"",
		"WITH-UPPERCASE",
		"has space",
		"has/slash",
		"_leading-underscore",
		"-leading-dash",
		"trailing space ",
		"semi;drop",
		"unicode-äö",
		"path/../traversal",
	}
	// Also reject something longer than 64 chars total (the regex limit).
	longOne := ""
	for i := 0; i < 65; i++ {
		longOne += "a"
	}
	bad = append(bad, longOne)

	for _, c := range bad {
		if validCategory(c) {
			t.Errorf("expected %q to be rejected", c)
		}
	}
}
