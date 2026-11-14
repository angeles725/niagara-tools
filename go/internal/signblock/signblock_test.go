package signblock

import (
	"strings"
	"testing"
)

// TestPartsAreTheTable pins Parts entry by entry and in order: the names the rows print and the patterns,
// as the retired shell table (toolbelt/lib/signing-block.sh SIGN_BLOCK_NAME and SIGN_BLOCK_ERE) held them.
// Build-module's dev-key WARN reads entry 0; lint-structure L15 and preflight sample-sign read them all.
func TestPartsAreTheTable(t *testing.T) {
	want := [][2]string{
		{"allowDefaultProfile.set(false) (or = false)", `allowDefaultProfile[[:space:]]*(\.set\([[:space:]]*false[[:space:]]*\)|=[[:space:]]*false([^[:alnum:]_]|$))`},
		{"niagaraSigning {", `niagaraSigning[[:space:]]*\{`},
		{"aliases", `(^|[^[:alnum:]_])aliases([^[:alnum:]_]|$)`},
		{"signingProfileFile", `signingProfileFile`},
	}
	if len(Parts) != len(want) {
		t.Fatalf("Parts has %d entries, want %d", len(Parts), len(want))
	}
	for k, p := range Parts {
		if p.Name != want[k][0] || p.Re.String() != want[k][1] {
			t.Errorf("Parts[%d] = %q %q, want %q %q", k, p.Name, p.Re, want[k][0], want[k][1])
		}
	}
}

// TestPartsMatchCode: what each pattern accepts and what it must not (the cases the shell table's consumers
// were tested with: both spellings of the switch, a name that only contains another).
func TestPartsMatchCode(t *testing.T) {
	cases := []struct {
		part int
		line string
		want bool
	}{
		{0, "allowDefaultProfile.set(false)", true},
		{0, "allowDefaultProfile . set( false )", false},
		{0, "allowDefaultProfile  .set( false )", true},
		{0, "allowDefaultProfile = false", true},
		{0, "allowDefaultProfile=false;", true},
		{0, "allowDefaultProfile = falsey", false},
		{0, "allowDefaultProfile.set(true)", false},
		{1, "niagaraSigning {", true},
		{1, "niagaraSigning{", true},
		{1, "niagaraSigning", false},
		{2, "aliases.set(listOf(\"k\"))", true},
		{2, "my_aliases", false},
		{2, "aliases2", false},
		{3, "signingProfileFile.set(file(\"p\"))", true},
	}
	for _, c := range cases {
		if got := Parts[c.part].Re.MatchString(c.line); got != c.want {
			t.Errorf("%s on %q = %v, want %v", Parts[c.part].Name, c.line, got, c.want)
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
