package retrodeltas

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// retro is a retro with the given header count, Delta count and Proposed kit deltas rows.
func retro(header, count string, rows ...string) string {
	return "<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: " + header + " -->\n# 2026-10-27 · kit · x\n\n" +
		"**Delta count**: " + count + "\n\n## Proposed kit deltas (propose-never-apply)\n\n| # | Delta | Target | Evidence |\n|---|---|---|---|\n" +
		strings.Join(rows, "\n") + "\n\n## Lessons\n\n| 9 | not a delta row |\n"
}

const index = "# Retro index\n\n| Retro | Module | Date | Status | Deltas |\n|---|---|---|---|---|\n"

func setup(t *testing.T, files map[string]string) string {
	t.Helper()
	d := t.TempDir()
	for n, b := range files {
		if err := os.WriteFile(filepath.Join(d, n), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func run(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

// TestCountRetroDeltas: the G39 contract (ledger.d/2026-10-23-r8-p1-process.md): rows, Delta count, header and
// INDEX row agree -> 0; any of them differing -> 1; usage and an unreadable retro or INDEX -> 3.
func TestCountRetroDeltas(t *testing.T) {
	two := []string{"| Δ1 | a | b | c |", "| 2 | d | e | f |"}
	for _, tc := range []struct {
		name   string
		files  map[string]string
		code   int
		out    string
		errHas string
	}{
		{"a 2-row retro declaring 2 everywhere", map[string]string{"r.md": retro("2", "2", two...), "INDEX.md": index + "| r.md | kit | 2026-10-27 | pending | 2 |\n"}, 0,
			"r.md  rows=2  count=2  header=2  index=2  OK\nTOTAL  rows=2  files=1  mismatches=0\n", ""},
		// Mutation: CRD-count -- ignoring the Delta count line passes a retro that declares 3 and lists 2
		{"declaring 3 for 2 rows", map[string]string{"r.md": retro("2", "3", two...), "INDEX.md": index + "| r.md | kit | 2026-10-27 | pending | 2 |\n"}, 1,
			"r.md  rows=2  count=3  header=2  index=2  MISMATCH\nTOTAL  rows=2  files=1  mismatches=1\n", ""},
		// Mutation: CRD-index -- skipping the INDEX row passes a generated index that says 1
		{"INDEX row 1 vs 2 rows", map[string]string{"r.md": retro("2", "2", two...), "INDEX.md": index + "| r.md | kit | 2026-10-27 | pending | 1 |\n"}, 1,
			"r.md  rows=2  count=2  header=2  index=1  MISMATCH\nTOTAL  rows=2  files=1  mismatches=1\n", ""},
		{"a note column after Deltas", map[string]string{"r.md": retro("2", "2", two...), "INDEX.md": "| Retro | Module | Date | Status | Deltas | Note |\n|---|---|---|---|---|---|\n| r.md | kit | 2026-10-27 | pending | 2 | <!-- 3 notes --> |\n"}, 0,
			"r.md  rows=2  count=2  header=2  index=2  OK\nTOTAL  rows=2  files=1  mismatches=0\n", ""},
		// Mutation: CRD-heading -- reading only "## Proposed kit deltas" counts 0 rows in a retro headed "## Deltas"
		{"a delta section headed ## Deltas", map[string]string{"r.md": strings.Replace(retro("2", "2", two...), "## Proposed kit deltas (propose-never-apply)", "## Deltas", 1), "INDEX.md": index + "| r.md | kit | 2026-10-27 | pending | 2 |\n"}, 0,
			"r.md  rows=2  count=2  header=2  index=2  OK\nTOTAL  rows=2  files=1  mismatches=0\n", ""},
		// Mutation: CRD-header -- dropping the header from the comparison passes a retro whose header alone says 3
		{"only the deltas header disagrees", map[string]string{"r.md": retro("3", "2", two...), "INDEX.md": index + "| r.md | kit | 2026-10-27 | pending | 2 |\n"}, 1,
			"r.md  rows=2  count=2  header=3  index=2  MISMATCH\nTOTAL  rows=2  files=1  mismatches=1\n", ""},
		{"no INDEX row for the retro", map[string]string{"r.md": retro("2", "2", two...), "INDEX.md": index}, 1,
			"r.md  rows=2  count=2  header=2  index=-  MISMATCH\nTOTAL  rows=2  files=1  mismatches=1\n", ""},
		{"no INDEX.md beside the retro", map[string]string{"r.md": retro("2", "2", two...)}, 3, "", "no INDEX.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := setup(t, tc.files)
			code, out, errs := run(filepath.Join(d, "r.md"))
			out = strings.ReplaceAll(out, d+"/", "")
			if code != tc.code || out != tc.out || !strings.Contains(errs, tc.errHas) || tc.errHas == "" && errs != "" {
				t.Errorf("got (%d, %q, %q), want (%d, %q, ~%q)", code, out, errs, tc.code, tc.out, tc.errHas)
			}
		})
	}
}

// TestTotalOverSeveralRetros: the TOTAL line sums the rows of every retro given, counts the files, and counts each
// mismatching retro once; mismatches among agreeing retros are exit 1. Retros in two directories each read their own
// INDEX.md (d.md's row exists only in other/INDEX.md, and says 2 there).
// Mutation: CRD-total -- keeping only the last retro's rows prints TOTAL rows=1
func TestTotalOverSeveralRetros(t *testing.T) {
	one := []string{"| Δ1 | a | b | c |"}
	three := []string{"| Δ1 | a | b | c |", "| Δ2 | d | e | f |", "| 3 | g | h | i |"}
	d := setup(t, map[string]string{
		"a.md":     retro("1", "1", one...),
		"b.md":     retro("3", "3", three...),
		"c.md":     retro("1", "2", one...),
		"INDEX.md": index + "| a.md | kit | 2026-10-27 | pending | 1 |\n| b.md | kit | 2026-10-27 | pending | 3 |\n| c.md | kit | 2026-10-27 | pending | 1 |\n",
	})
	sub := filepath.Join(d, "other")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for n, b := range map[string]string{"d.md": retro("1", "1", one...), "INDEX.md": index + "| d.md | kit | 2026-10-27 | pending | 2 |\n"} {
		if err := os.WriteFile(filepath.Join(sub, n), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, out, errs := run(filepath.Join(d, "a.md"), filepath.Join(d, "b.md"), filepath.Join(d, "c.md"), filepath.Join(sub, "d.md"))
	out = strings.ReplaceAll(out, d+"/", "")
	want := "a.md  rows=1  count=1  header=1  index=1  OK\n" +
		"b.md  rows=3  count=3  header=3  index=3  OK\n" +
		"c.md  rows=1  count=2  header=1  index=1  MISMATCH\n" +
		"other/d.md  rows=1  count=1  header=1  index=2  MISMATCH\n" +
		"TOTAL  rows=6  files=4  mismatches=2\n"
	if code != 1 || out != want || errs != "" {
		t.Errorf("got (%d, %q, %q), want (1, %q, \"\")", code, out, errs, want)
	}
}

// TestUsage: no argument and an unreadable retro are exit 3.
func TestUsage(t *testing.T) {
	if code, _, errs := run(); code != 3 || !strings.HasPrefix(errs, "usage: count-retro-deltas") {
		t.Errorf("no argument: %d %q", code, errs)
	}
	d := setup(t, map[string]string{"INDEX.md": index})
	if code, _, errs := run(filepath.Join(d, "missing.md")); code != 3 || !strings.Contains(errs, "cannot read") {
		t.Errorf("missing retro: %d %q", code, errs)
	}
}
