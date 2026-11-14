package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// TestProfileSelection: which profiles gradle builds.
func TestProfileSelection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		setup    func(f *fixture)
		flags    []string
		profiles string // the profiles gradle builds, space separated; "" when the chain stops with exit 10
		output   string
	}{
		{"sources decide; a scaffold is skipped", nil, nil, "rt ux", "==> skipping Foo-wb: scaffold without sources (pass --profiles to force)\n"},
		{"--profiles replaces detection, in its order", nil, []string{"--profiles", "wb,rt"}, "wb rt", ""},
		{"a trailing comma adds no profile", nil, []string{"--profiles", "rt,"}, "rt", ""},
		{"a profile dir that does not exist", nil, []string{"--profiles", "rt,se"}, "", "build.sh: profile dir missing: "},
		{"build.gradle counts as a gradle file", func(f *fixture) {
			os.Remove(f.root + "/Foo/Foo-rt/Foo-rt.gradle.kts")
			f.write(f.root+"/Foo/Foo-rt/build.gradle", "")
		}, nil, "rt ux", ""},
		{"sources without a gradle file are a scaffold", func(f *fixture) { os.Remove(f.root + "/Foo/Foo-rt/Foo-rt.gradle.kts") }, nil, "ux", "==> skipping Foo-rt: scaffold"},
		{"an html file is a source", func(f *fixture) { f.write(f.root+"/Foo/Foo-wb/src/rc/v.html", "") }, nil, "rt ux wb", ""},
		{"no profile has sources", func(f *fixture) {
			os.RemoveAll(f.root + "/Foo/Foo-rt/src/com")
			os.RemoveAll(f.root + "/Foo/Foo-ux/src/rc")
		}, nil, "", "build.sh: no buildable profile under "},
	}
	for _, c := range cases {
		f := newFixture(t)
		if c.setup != nil {
			c.setup(f)
		}
		code := f.build(c.flags...)
		var built []string
		for _, arg := range strings.Split(f.call("gradlew"), " ") {
			if p, ok := strings.CutSuffix(arg, ":jar>"); ok {
				built = append(built, strings.TrimPrefix(p, "<:Foo-"))
			}
		}
		wantCode := ExitOK
		if c.profiles == "" {
			wantCode = ExitEnv
		}
		f.wants(c.name, code, wantCode, c.output)
		if got := strings.Join(built, " "); got != c.profiles {
			t.Errorf("%s: gradle built %q, want %q", c.name, got, c.profiles)
		}
	}
}
