package buildmodule

import (
	"strings"
	"testing"
)

// TestMirrorSigningWarning: every mirror build says up front, before gradle, that its jar is signed with the
// mirror home's throwaway key and cannot be installed, and names the build that works; a real home gets none.
func TestMirrorSigningWarning(t *testing.T) {
	const throwaway = "build.sh: WARN — mirror build: the jar is signed with the throwaway key of the mirror user home"
	cases := []struct {
		name   string
		mirror bool
		gradle int // exit code of the fake gradlew
		code   int
		warned bool
	}{
		{"a mirror build", true, 0, ExitOK, true},
		{"a mirror build whose gradle fails", true, 1, ExitGradle, true},
		{"a real home", false, 0, ExitOK, false},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.signs()
		if c.mirror {
			f.write(f.home+"/"+mirrorMarker, "")
		}
		if c.gradle != 0 {
			f.does("gradlew", "exit 1\n")
		}
		got := f.run(f.root, "Foo", f.home)
		if !c.mirror {
			got = f.build()
		}
		f.wants(c.name, got, c.code)
		if has := strings.Contains(f.err, throwaway); has != c.warned {
			t.Errorf("%s: throwaway-key warning = %v, want %v\nstderr: %s", c.name, has, c.warned, f.err)
		}
		if !c.warned {
			continue
		}
		for _, text := range []string{"cannot be installed", "-Pniagara_home=" + f.home, "real HOME", f.home + "/.mirror-user-home"} {
			if !strings.Contains(f.err, text) {
				t.Errorf("%s: warning lacks %q\nstderr: %s", c.name, text, f.err)
			}
		}
	}
}
