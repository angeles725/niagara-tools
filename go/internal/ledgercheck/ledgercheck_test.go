package ledgercheck

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// template is the kit's own templates/VERSION-LEDGER.md: the contract every client ledger is checked against.
var template = filepath.Join("..", "..", "..", "build-n4-module-kit", "templates", "VERSION-LEDGER.md")

const header = "| Module | Version | Date | Status | What changed | Defect found | Root cause | Class | Resolved by | Prevention (retro + Δ) |"

// ledger is a client ledger in the template's shape with the given Versions rows.
func ledger(rows ...string) string {
	return "# VERSION-LEDGER — client A\n\n## Versions\n\n" + header + "\n|---|---|---|---|---|---|---|---|---|---|\n" +
		strings.Join(rows, "\n") + "\n\n## Root-cause classes\n\n" +
		"| Class | Description | Typical new work that should check it |\n|---|---|---|\n" +
		"| 1 | Framework rule unknown | a new facet |\n| 5 | Fail-open UX | a new indicator |\n\n## Open items\n\n- none\n"
}

func run(t *testing.T, body string, args ...string) (int, string, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "VERSION-LEDGER.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	env := map[string]string{TemplateEnv: template}
	code := Main(append(args, p), cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
	return code, strings.ReplaceAll(o.String(), p, "L.md"), e.String()
}

const good = "| Pan | 1.2.0 | 2026-10-01 | deployed | first cut | — | — | — | — | — |"

func TestMain(t *testing.T) {
	cases := []struct {
		name, body string
		code       int
		out        string
	}{
		{"a clean ledger", ledger(good, "| Pan | 1.2.1 | 2026-10-02 | built | fix | blank HMI | no watchdog | 5 | 1.2.2 | retro x Δ1 |"), 0, ""},
		{"an empty Versions table (a fresh scaffold)", ledger(), 0, ""},
		// Mutation: LC1-title -- a file with another title is not a version ledger
		{"no title", strings.Replace(ledger(good), "# VERSION-LEDGER — client A", "# Notes", 1), 1,
			"FAIL  ledger-check  L.md:1  LC1: the first line is not a `# VERSION-LEDGER` title\n"},
		// Mutation: LC2-section -- a ledger without its Open items section loses the per-client aggregate
		{"a missing section", strings.Replace(ledger(good), "## Open items", "## Later", 1), 1,
			"FAIL  ledger-check  L.md  LC2: no `## Open items` section (templates/VERSION-LEDGER.md)\n"},
		// Mutation: LC3-header -- a reordered header reads every row's cells into the wrong columns
		{"a Versions header that is not the template's", strings.Replace(ledger(good), "| Date | Status |", "| Status | Date |", 1), 1,
			"FAIL  ledger-check  L.md:5  LC3: the Versions table header differs from the template's (" + header + ")\n"},
		{"a row with too few cells", ledger("| Pan | 1.2.0 | 2026-10-01 | built |"), 1,
			"FAIL  ledger-check  L.md:7  LC4: row has 4 cells, want 10\n"},
		{"an empty version", ledger("| Pan |  | 2026-10-01 | built | x | — | — | — | — | — |"), 1,
			"FAIL  ledger-check  L.md:7  LC4: empty Module or Version cell\n"},
		// Mutation: LC5-status -- a free-text status hides a jar that was never deployed
		{"a status outside the template's domain", ledger("| Pan | 1.2.0 | 2026-10-01 | installed | x | — | — | — | — | — |"), 1,
			"FAIL  ledger-check  L.md:7  LC5: status installed is not one of built, packaged, deployed, superseded, do-not-install\n"},
		{"a date that is not a date", ledger("| Pan | 1.2.0 | 2026-13-01 | built | x | — | — | — | — | — |"), 1,
			"FAIL  ledger-check  L.md:7  LC6: date 2026-13-01 is not a YYYY-MM-DD date\n"},
		// Mutation: LC7-class -- a class the ledger does not declare escapes the repeated-class escalation
		{"an undeclared class", ledger("| Pan | 1.2.0 | 2026-10-01 | built | x | bug | cause | 9 | 1.2.1 | — |"), 1,
			"FAIL  ledger-check  L.md:7  LC7: class 9 is not in the Root-cause classes table\n"},
		// Mutation: LC8-defect -- a defect with no class never counts toward a repeated class
		{"a defect with no root cause or class", ledger("| Pan | 1.2.0 | 2026-10-01 | built | x | blank HMI | — | — | — | — |"), 1,
			"FAIL  ledger-check  L.md:7  LC8: a defect found with no Root cause and Class\n"},
		// Mutation: LC9-placeholder -- the template's sample row left in place reads as a real version
		{"the template's placeholder row", ledger("| <Module> | <x.y.z> | <YYYY-MM-DD> | built | <one line> | — | — | — | — | — |"), 0,
			"WARN  ledger-check  L.md:7  LC9: template placeholder left in the row (replace or delete it)\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errs := run(t, tt.body)
			if code != tt.code || out != tt.out || errs != "" {
				t.Errorf("got (%d, %q, %q)\nwant (%d, %q)", code, out, errs, tt.code, tt.out)
			}
		})
	}
}

// TestTemplateItself: the kit's template passes with only its sample row's placeholder WARN, so the contract is
// read from it and never drifts from it.
func TestTemplateItself(t *testing.T) {
	b, err := os.ReadFile(template)
	if err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(t, string(b))
	if code != 0 || strings.Count(out, "\n") != 1 || !strings.Contains(out, "LC9:") || errs != "" {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}

// TestEnv: usage, an unreadable ledger and a missing or malformed template are exit 3, never a clean pass.
func TestEnv(t *testing.T) {
	var o, e bytes.Buffer
	none := cli.Stdio{Out: &o, Err: &e, Getenv: func(string) string { return "" }}
	if code := Main(nil, none); code != 3 || !strings.HasPrefix(e.String(), "usage: ledger-check.sh") {
		t.Errorf("no argument: %d %q", code, e.String())
	}
	e.Reset()
	if code := Main([]string{"x.md"}, none); code != 3 || !strings.Contains(e.String(), "no template") {
		t.Errorf("no template: %d %q", code, e.String())
	}
	if code, _, errs := run(t, ledger(good), "--template", filepath.Join(t.TempDir(), "none.md")); code != 3 || !strings.Contains(errs, "cannot read template") {
		t.Errorf("unreadable template: %d %q", code, errs)
	}
	bad := filepath.Join(t.TempDir(), "T.md")
	if err := os.WriteFile(bad, []byte("# VERSION-LEDGER\n\nno tables\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := run(t, ledger(good), "--template", bad); code != 3 || !strings.Contains(errs, "template has no Versions table") {
		t.Errorf("malformed template: %d %q", code, errs)
	}
	e.Reset()
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(string) string { return template }}
	if code := Main([]string{filepath.Join(t.TempDir(), "missing.md")}, s); code != 3 || !strings.Contains(e.String(), "cannot read") {
		t.Errorf("missing ledger: %d %q", code, e.String())
	}
}
