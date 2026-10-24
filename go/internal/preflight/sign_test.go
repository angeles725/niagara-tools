package preflight

import (
	"strings"
	"testing"
)

func TestKtsCode(t *testing.T) {
	d := t.TempDir()
	src := "a // c\nb /* c */ d /* e\nf */ g\n\"s // x\" h \"\\\"/*\" i\n\"\"\"raw\n/* \"\"\" j\nk \"open\n"
	mk(t, d, "f="+src)
	want := []string{"a ", "b  d ", " g", `"" h "" i`, `"""`, `""" j`, `k "`}
	if got := ktsCode(d + "/f"); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("ktsCode =\n%q\nwant\n%q", got, want)
	}
	if got := ktsCode(d + "/missing"); got != nil {
		t.Errorf("missing file: %q, want no code", got)
	}
}
