package foldaudit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// run calls Main with buffers and an empty environment except for arg0.
func run(arg0 string, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(name string) string {
		if name == Arg0Env {
			return arg0
		}
		return ""
	}}
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
		"fold-audit: 5 folded, 2 cited, 3 uncited\n"
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

// Bash-compatible edges of the INDEX loop (each captured from the bash tool).
func TestMainBashEdges(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"core.md": "[ev: retro tail-row-x Δ1] [ev: retro nul-row-x]\n",
		"retros/INDEX.md": "| 2026-10-01-octal.md | folded | 010 |\n" +
			"| 2026-10-01-badoct.md | folded | 09 |\n" +
			"| 2026-10-01-n\x00ul-row-x.md | fol\x00ded | 0 |\n" +
			"| 2026-10-01-tail-row-x.md | folded | 1 |", // no trailing newline: bash `read` never runs the body
	})
	code, out, errs := run("/x/sweep-fold-audit.sh", "--strict", "--deltas-since", "2026-01-01", root+"/retros/INDEX.md", root)
	var want strings.Builder
	for k := 1; k <= 8; k++ {
		want.WriteString(w + "2026-10-01-octal.md Δ" + string(rune('0'+k)) + " not cited\n")
	}
	want.WriteString(w + "2026-10-01-octal.md" + nocit)
	want.WriteString(w + "2026-10-01-badoct.md" + nocit)
	want.WriteString("fold-audit: 3 folded, 1 cited, 2 uncited\n")
	want.WriteString("fold-audit: deltas: 8 checked, 8 not cited (folded rows since 2026-01-01)\n")
	wantErr := "/x/sweep-fold-audit.sh: line 132: ((: 09: value too great for base (error token is \"09\")\n"
	if code != ExitStrict || out != want.String() || errs != wantErr {
		t.Errorf("got (%d, %q, %q)\nwant (1, %q, %q)", code, out, errs, want.String(), wantErr)
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
	if code != ExitOK || out != "fold-audit: 0 folded, 0 cited, 0 uncited\n" ||
		errs != "sweep-fold-audit.sh: line 152: "+index+": Permission denied\n" {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}
