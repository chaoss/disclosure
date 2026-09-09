package cmd

import "testing"

func TestVersionFrom(t *testing.T) {
	cases := []struct {
		name     string
		override string
		build    string
		want     string
	}{
		{"ldflags override wins", "1.2.3", "v9.9.9", "1.2.3"},
		{"dev falls back to build info", "dev", "v1.0.1", "v1.0.1"},
		{"empty override falls back to build info", "", "v2.0.0", "v2.0.0"},
		{"devel build info is ignored", "dev", "(devel)", "dev"},
		{"empty build info falls back to dev", "dev", "", "dev"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := versionFrom(c.override, func() string { return c.build })
			if got != c.want {
				t.Errorf("versionFrom(%q, ()=>%q) = %q, want %q", c.override, c.build, got, c.want)
			}
		})
	}
}
