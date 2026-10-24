package preflight

import (
	"os"
	"regexp"
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

// TestSignBlockMatchesTheShellTable pins signBlock to toolbelt/lib/signing-block.sh entry by entry: the
// bash build.sh dev-key WARN and the Go lint-structure L15 read the same table.
func TestSignBlockMatchesTheShellTable(t *testing.T) {
	b, err := os.ReadFile("../../../build-n4-module-kit/toolbelt/lib/signing-block.sh")
	if err != nil {
		t.Fatal(err)
	}
	quoted := regexp.MustCompile(`'([^']*)'`)
	array := func(name string) []string {
		m := regexp.MustCompile(`(?s)\n` + name + `=\((.*?)\)\n`).FindSubmatch(b)
		if m == nil {
			t.Fatalf("no %s=( … ) in signing-block.sh", name)
		}
		var out []string
		for _, q := range quoted.FindAllSubmatch(m[1], -1) {
			out = append(out, string(q[1]))
		}
		return out
	}
	eres, names := array("SIGN_BLOCK_ERE"), array("SIGN_BLOCK_NAME")
	if len(eres) != len(signBlock) || len(names) != len(signBlock) {
		t.Fatalf("shell table has %d EREs and %d names, signBlock %d entries", len(eres), len(names), len(signBlock))
	}
	for k, part := range signBlock {
		if part.name != names[k] || part.re.String() != eres[k] {
			t.Errorf("signBlock[%d] = %q %q, shell %q %q", k, part.name, part.re, names[k], eres[k])
		}
	}
}
