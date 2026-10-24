package signblock

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestPartsMatchTheShellTable pins Parts to toolbelt/lib/signing-block.sh entry by entry: the bash build.sh
// dev-key WARN reads the shell table, lint-structure L15 and preflight sample-sign read Parts.
func TestPartsMatchTheShellTable(t *testing.T) {
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
	if len(eres) != len(Parts) || len(names) != len(Parts) {
		t.Fatalf("shell table has %d EREs and %d names, Parts %d entries", len(eres), len(names), len(Parts))
	}
	for k, p := range Parts {
		if p.Name != names[k] || p.Re.String() != eres[k] {
			t.Errorf("Parts[%d] = %q %q, shell %q %q", k, p.Name, p.Re, names[k], eres[k])
		}
	}
}

func TestMissing(t *testing.T) {
	code := []string{"allowDefaultProfile = false", "niagaraSigning {", "  my_aliases signingProfileFile"}
	if got := strings.Join(Missing(code), "|"); got != "aliases" {
		t.Errorf("Missing = %q, want aliases (my_aliases is another name)", got)
	}
	if got := len(Missing(nil)); got != len(Parts) {
		t.Errorf("no code: %d missing, want all %d", got, len(Parts))
	}
}
