package foldaudit

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// run calls Main with buffers and an empty environment.
func run(_ string, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(string) string { return "" }}
	code = Main(args, s)
	return code, o.String(), e.String()
}

// fixture is tests/fixtures/fold-audit, shared with tests/sweep-fold-audit.bats (F1-F6).
func fixture(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../../tests/fixtures/fold-audit")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// kit writes a kit root from rel-path -> content.
func kit(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, text := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const (
	w     = "fold-audit: WARN "
	nocit = " folded with no [ev: retro …] citation\n"
)

// F1-F6: the stem-citation fixture.
func TestMainStemFixture(t *testing.T) {
	t.Parallel()
	dir := fixture(t)
	want := w + "2026-01-01-orphan-report.md" + nocit +
		w + "2026-01-03-self-ref-only.md" + nocit +
		w + "2026-01-06-detail-render-doors.md" + nocit +
		w + dir + `/core.md:8 retro alias "5rooms" — use the exact stem "dashboardpan-5rooms" (CONTRIBUTING.md:261)` + "\n" +
		"fold-audit: 5 folded, 2 cited, 3 uncited\n" +
		"fold-audit: aliases: 1 tokens are not an exact stem (0 ambiguous)\n"
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{dir + "/INDEX.md", dir}, ExitOK},
		{[]string{"--strict", dir + "/INDEX.md", dir}, ExitStrict},
	} {
		code, out, errs := run("", c.args...)
		if code != c.code || out != want || errs != "" {
			t.Errorf("%q: got (%d, %q, %q), want (%d, %q, \"\")", c.args, code, out, errs, c.code, want)
		}
	}
}

// deltaKit is the _delta_fixture of tests/sweep-fold-audit.bats (F7-F9).
func deltaKit(t *testing.T) string {
	return kit(t, map[string]string{
		"retros/INDEX.md": "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n" +
			"| 2026-09-01-old-style.md | kit | 2026-09-01 | folded | 2 |\n" +
			"| 2026-10-01-full.md | kit | 2026-10-01 | folded | 2 |\n" +
			"| 2026-10-01-gap.md | kit | 2026-10-01 | folded | 3 |\n" +
			"| 2026-10-01-deferred.md | kit | 2026-10-01 | folded | 2 |\n" +
			"| 2026-10-02-campaign-close.md | kit | 2026-10-02 | folded | 1 |\n" +
			"| 2026-10-02-one.md | kit | 2026-10-02 | folded | 1 |\n" +
			"| 2026-10-02-later.md | kit | 2026-10-02 | pending | 4 |\n",
		"core.md": "Doc [ev: retro old-style] cites by stem only (before the cutoff).\n" +
			"Rules [ev: retro full Δ1] and [ev: retro full Δ2].\n" +
			"Rule [ev: retro gap Δ1] and [ev: retro gap Δ3].\n" +
			"Rule [ev: retro deferred Δ1]; out of repo [deferred: retro deferred Δ2].\n" +
			"Rule [ev: retro retro-campaign-close Δ1] names another stem.\n" +
			"Rule [ev: retro one Δ10] is not Δ1.\n",
	})
}

// F7-F9: Δ-granular reconcile (output captured from the bash tool).
func TestMainDeltas(t *testing.T) {
	t.Parallel()
	root := deltaKit(t)
	want := w + "2026-10-01-full.md" + nocit +
		w + "2026-10-01-gap.md Δ2 not cited\n" +
		w + "2026-10-01-gap.md" + nocit +
		w + "2026-10-02-campaign-close.md Δ1 not cited\n" +
		w + "2026-10-02-campaign-close.md" + nocit +
		w + "2026-10-02-one.md Δ1 not cited\n" +
		w + "2026-10-02-one.md" + nocit +
		"fold-audit: 6 folded, 2 cited, 4 uncited\n" +
		"fold-audit: deltas: 9 checked, 3 not cited (folded rows since 2026-09-24)\n"
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{"--deltas-since", "2026-09-24", root + "/retros/INDEX.md", root}, ExitOK},
		{[]string{"--strict", "--deltas-since", "2026-09-24", root + "/retros/INDEX.md", root}, ExitStrict},
	} {
		code, out, errs := run("", c.args...)
		if code != c.code || out != want || errs != "" {
			t.Errorf("%q: got (%d, %q, %q), want (%d, %q, \"\")", c.args, code, out, errs, c.code, want)
		}
	}
}

// F11: a ledger.d/ fragment is corpus.
func TestMainLedgerFragment(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"retros/INDEX.md":          "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n| 2026-10-22-ledger-only.md | kit | 2026-10-22 | folded | 1 |\n",
		"ledger.d/2026-10-22-x.md": "<!-- ledger: kit -->\n<!-- date: 2026-10-22 -->\nlog: 2026-10-22 · folded [ev: retro ledger-only Δ1]\n",
	})
	code, out, errs := run("", "--strict", "--deltas-since", "2026-09-24", root+"/retros/INDEX.md", root)
	want := "fold-audit: 1 folded, 1 cited, 0 uncited\nfold-audit: deltas: 1 checked, 0 not cited (folded rows since 2026-09-24)\n"
	if code != ExitOK || out != want || errs != "" {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}

// F10 and the env checks: exit 3 with the bash messages.
func TestMainUsageAndEnv(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{"INDEX.md": ""})
	missing := filepath.Join(root, "nope")
	cases := []struct {
		args []string
		errs string
	}{
		{[]string{"--deltas-since", "2026-9-1", root + "/INDEX.md", root}, "fold-audit: --deltas-since needs a YYYY-MM-DD date\n" + usage + "\n"},
		{[]string{root + "/INDEX.md"}, usage + "\n"},
		{[]string{missing, root}, "fold-audit: INDEX not found: " + missing + "\n"},
		{[]string{root, root}, "fold-audit: INDEX not found: " + root + "\n"},
		{[]string{root + "/INDEX.md", missing}, "fold-audit: kit-root not found: " + missing + "\n"},
	}
	for _, c := range cases {
		code, out, errs := run("", c.args...)
		if code != ExitEnv || out != "" || errs != c.errs {
			t.Errorf("%q: got (%d, %q, %q), want (3, \"\", %q)", c.args, code, out, errs, c.errs)
		}
	}
}

// The fail-open cases of the bash tool, fixed in the port: a leading-zero deltas cell is decimal
// (bash read 010 as 8 and aborted on 09), an overflowing cell is unreadable, and an unterminated
// last INDEX row is audited (bash `while read` dropped it).
func TestMainFailClosedEdges(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"core.md": "[ev: retro tail-row-x] [ev: retro nul-row-x]\n",
		"retros/INDEX.md": "| 2026-10-01-octal.md | folded | 010 |\n" +
			"| 2026-10-01-badoct.md | folded | 09 |\n" +
			"| 2026-10-01-n\x00ul-row-x.md | fol\x00ded | 0 |\n" +
			"| 2026-10-01-huge.md | folded | 18446744073709551617 |\n" +
			"| 2026-10-01-tail-row-x.md | folded | 1 |",
	})
	code, out, errs := run("", "--strict", "--deltas-since", "2026-01-01", root+"/retros/INDEX.md", root)
	var want strings.Builder
	for k := 1; k <= 10; k++ {
		want.WriteString(w + "2026-10-01-octal.md Δ" + strconv.Itoa(k) + " not cited\n")
	}
	want.WriteString(w + "2026-10-01-octal.md" + nocit)
	for k := 1; k <= 9; k++ {
		want.WriteString(w + "2026-10-01-badoct.md Δ" + strconv.Itoa(k) + " not cited\n")
	}
	want.WriteString(w + "2026-10-01-badoct.md" + nocit)
	want.WriteString(w + "2026-10-01-huge.md deltas cell unreadable (18446744073709551617)\n")
	want.WriteString(w + "2026-10-01-huge.md" + nocit)
	want.WriteString(w + "2026-10-01-tail-row-x.md Δ1 not cited\n")
	want.WriteString("fold-audit: 5 folded, 2 cited, 3 uncited\n")
	// The unreadable cell has its own counter: it is not a "not cited" Δ (R2-unreadable-cell).
	want.WriteString("fold-audit: deltas: 20 checked, 20 not cited (folded rows since 2026-01-01)\n")
	want.WriteString("fold-audit: deltas: 1 unreadable cells\n")
	if code != ExitStrict || out != want.String() || errs != "" {
		t.Errorf("got (%d, %q, %q)\nwant (1, %q, \"\")", code, out, errs, want.String())
	}
}

func TestMainUnreadableIndex(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 file")
	}
	root := kit(t, map[string]string{"INDEX.md": "| 2026-01-01-a.md | folded | 1 |\n"})
	index := root + "/INDEX.md"
	if err := os.Chmod(index, 0); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run("", "--strict", index, root)
	if code != ExitEnv || out != "" || errs != "fold-audit: cannot read INDEX: open "+index+": permission denied\n" {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}

// An unreadable cell still fails --strict, and the unreadable line appears only when one exists.
func TestMainUnreadableCellCounter(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"core.md":         "[ev: retro word-cell] [ev: retro one-cell Δ1]\n",
		"retros/INDEX.md": "| 2026-10-01-word-cell.md | folded | x |\n| 2026-10-01-one-cell.md | folded | 1 |\n",
	})
	code, out, _ := run("", "--strict", "--deltas-since", "2026-01-01", root+"/retros/INDEX.md", root)
	want := w + "2026-10-01-word-cell.md deltas cell unreadable (x)\n" +
		"fold-audit: 2 folded, 2 cited, 0 uncited\n" +
		"fold-audit: deltas: 1 checked, 0 not cited (folded rows since 2026-01-01)\n" +
		"fold-audit: deltas: 1 unreadable cells\n"
	if code != ExitStrict || out != want {
		t.Errorf("got (%d, %q), want (1, %q)", code, out, want)
	}
}

// aliasKit: unique, ambiguous and alias-only citations next to exact stems (#419, CONTRIBUTING.md:261).
func aliasKit(t *testing.T) string {
	return kit(t, map[string]string{
		"retros/INDEX.md": "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n" +
			"| 2026-10-01-comppan-fase1-staging.md | kit | 2026-10-01 | folded | 1 |\n" +
			"| 2026-10-01-schema-risk-gate.md | kit | 2026-10-01 | folded | 0 |\n" +
			"| 2026-10-01-schema-risk-ledger.md | kit | 2026-10-01 | pending | 0 |\n" +
			"| 2026-10-22-go-sweep-fold-audit.md | kit | 2026-10-22 | folded | 0 |\n" +
			"| 2026-10-01-alias-only-row.md | kit | 2026-10-01 | folded | 1 |\n",
		"core.md": "Rule [ev: retro comppan-fase1-staging Δ1] cites the exact stem.\n" +
			"Short [ev: retro comppan-fase1] is a unique alias.\n" +
			"Gate [ev: retro schema-risk-gate]; [ev: retro schema-risk] has two candidates.\n" +
			"Tool [ev: retro fold-audit] and [ev: retro go-sweep-fold-audit].\n" +
			"Only [ev: retro alias-only] and [deferred: retro alias-only Δ1] cite the last row.\n" +
			"No stem [ev: retro unrelated-token], a placeholder [ev: retro X].\n",
	})
}

func TestMainAliases(t *testing.T) {
	t.Parallel()
	root := aliasKit(t)
	index := root + "/retros/INDEX.md"
	core := root + "/core.md"
	aliasRows := func(kind string) string {
		p := "fold-audit: " + kind + " " + core
		return p + `:2 retro alias "comppan-fase1" — use the exact stem "comppan-fase1-staging" (CONTRIBUTING.md:261)` + "\n" +
			p + `:3 retro alias "schema-risk" is ambiguous — use one exact stem of "schema-risk-gate" "schema-risk-ledger" (CONTRIBUTING.md:261)` + "\n" +
			p + `:4 retro alias "fold-audit" — use the exact stem "go-sweep-fold-audit" (CONTRIBUTING.md:261)` + "\n" +
			p + `:5 retro alias "alias-only" — use the exact stem "alias-only-row" (CONTRIBUTING.md:261)` + "\n"
	}
	const deltas = "fold-audit: deltas: 2 checked, 1 not cited (folded rows since 2026-09-24)\n"
	const aliasSum = "fold-audit: aliases: 4 tokens are not an exact stem (1 ambiguous)\n"
	// Default: aliases still credit (the NOTE case stays) and the alias rows are advisory WARN rows.
	wantDefault := "fold-audit: NOTE ambiguous citation token credits 2026-10-01-comppan-fase1-staging.md: comppan-fase1 comppan-fase1-staging\n" +
		"fold-audit: NOTE ambiguous citation token credits 2026-10-01-schema-risk-gate.md: schema-risk schema-risk-gate\n" +
		"fold-audit: NOTE ambiguous citation token credits 2026-10-22-go-sweep-fold-audit.md: fold-audit go-sweep-fold-audit\n" +
		w + "2026-10-01-alias-only-row.md Δ1 not cited\n" +
		aliasRows("WARN") +
		"fold-audit: 4 folded, 4 cited, 0 uncited\n" + deltas + aliasSum
	// --exact-stems: only an exact stem credits, so the alias-only row is uncited and every alias is a FAIL row.
	wantExact := w + "2026-10-01-alias-only-row.md Δ1 not cited\n" +
		w + "2026-10-01-alias-only-row.md" + nocit +
		aliasRows("FAIL") +
		"fold-audit: 4 folded, 3 cited, 1 uncited\n" + deltas + aliasSum
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--deltas-since", "2026-09-24", index, root}, wantDefault},
		{[]string{"--exact-stems", "--deltas-since", "2026-09-24", index, root}, wantExact},
	} {
		code, out, errs := run("", c.args...)
		if code != ExitOK || out != c.want || errs != "" {
			t.Errorf("%q: got (%d, %q, %q)\nwant (0, %q, \"\")", c.args, code, out, errs, c.want)
		}
	}
}

// The alias rows are advisory by default and fail --strict only under --exact-stems.
func TestMainAliasExit(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"retros/INDEX.md": "| 2026-10-01-exact-stem-row.md | folded | 0 |\n",
		"core.md":         "[ev: retro exact-stem-row] and the alias [ev: retro stem-row]\n",
	})
	index := root + "/retros/INDEX.md"
	row := `:1 retro alias "stem-row" — use the exact stem "exact-stem-row" (CONTRIBUTING.md:261)` + "\n"
	tail := "fold-audit: 1 folded, 1 cited, 0 uncited\nfold-audit: aliases: 1 tokens are not an exact stem (0 ambiguous)\n"
	for _, c := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"--strict", index, root}, ExitOK, "fold-audit: NOTE ambiguous citation token credits 2026-10-01-exact-stem-row.md: exact-stem-row stem-row\nfold-audit: WARN " + root + "/core.md" + row + tail},
		// Mutation: F21 -- counting a FAIL alias row as advisory lets --exact-stems --strict exit 0.
		{[]string{"--strict", "--exact-stems", index, root}, ExitStrict, "fold-audit: FAIL " + root + "/core.md" + row + tail},
		{[]string{"--exact-stems", index, root}, ExitOK, "fold-audit: FAIL " + root + "/core.md" + row + tail},
	} {
		code, out, errs := run("", c.args...)
		if code != c.code || out != c.want || errs != "" {
			t.Errorf("%q: got (%d, %q, %q)\nwant (%d, %q, \"\")", c.args, code, out, errs, c.code, c.want)
		}
	}
}
