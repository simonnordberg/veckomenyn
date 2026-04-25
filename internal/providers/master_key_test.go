package providers

import "testing"

func TestDecideKey(t *testing.T) {
	cases := []struct {
		name             string
		envSet, dbSet    bool
		envEqualsDB      bool
		want             keyChoice
	}{
		{"both empty -> generate", false, false, false, choiceGenerate},
		{"env only -> mirror", true, false, false, choiceMirrorEnvToDB},
		{"db only -> use db", false, true, false, choiceUseDB},
		{"both set, equal -> use env", true, true, true, choiceUseEnv},
		{"both set, differ -> mismatch", true, true, false, choiceMismatchError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideKey(c.envSet, c.dbSet, c.envEqualsDB)
			if got != c.want {
				t.Errorf("decideKey(env=%v, db=%v, eq=%v) = %v, want %v",
					c.envSet, c.dbSet, c.envEqualsDB, got, c.want)
			}
		})
	}
}
