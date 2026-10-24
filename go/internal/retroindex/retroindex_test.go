package retroindex

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/angeles725/niagara-tools/go/internal/gen"
)

const (
	errStatus = "no <!-- review-status: pending|folded --> marker in lines 1-5"
	errModule = "needs exactly one <!-- module: <Module|kit> --> header in lines 1-10"
	errDeltas = "needs exactly one <!-- deltas: <N> --> header (N an integer) in lines 1-10"
	errDate   = "no YYYY-MM-DD date in the filename"
)

func TestParseHeader(t *testing.T) {
	t.Parallel()
	ok := "<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 2 -->\n# x\n"
	cases := []struct {
		name, file, text, row, reason string
	}{
		{"valid", "2026-09-01-a.md", ok, "| 2026-09-01-a.md | kit | 2026-09-01 | pending | 2 |", ""},
		{"date anywhere in the name", "APPLY-PLAN-2026-09-03-x.md", ok, "| APPLY-PLAN-2026-09-03-x.md | kit | 2026-09-03 | pending | 2 |", ""},
		{"status is the first word after the colon", "2026-09-01-a.md",
			"<!--  review-status:\tfolded (B12) -->\n<!-- module: Pan -->\n<!-- deltas: 0 -->\n",
			"| 2026-09-01-a.md | Pan | 2026-09-01 | folded | 0 |", ""},
		{"index-comment is a trailing row comment", "2026-09-01-a.md",
			ok + "<!-- index-comment: disposition: parked — done -->\n",
			"| 2026-09-01-a.md | kit | 2026-09-01 | pending | 2 | <!-- disposition: parked — done -->", ""},
		{"no status marker", "2026-09-01-a.md", "<!-- module: kit -->\n<!-- deltas: 0 -->\n", "", errStatus},
		{"status marker below line 5", "2026-09-01-a.md",
			"\n\n\n\n\n<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 0 -->\n", "", errStatus},
		{"first status marker decides", "2026-09-01-a.md",
			"<!-- review-status: maybe -->\n<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 0 -->\n", "", errStatus},
		{"status glued to the closer", "2026-09-01-a.md",
			"<!-- review-status: pending-->\n<!-- module: kit -->\n<!-- deltas: 0 -->\n", "", errStatus},
		{"no module", "2026-09-01-a.md", "<!-- review-status: pending -->\n<!-- deltas: 0 -->\n", "", errModule},
		{"two modules", "2026-09-01-a.md",
			"<!-- review-status: pending -->\n<!-- module: a -->\n<!-- module: b -->\n<!-- deltas: 0 -->\n", "", errModule},
		{"empty module", "2026-09-01-a.md", "<!-- review-status: pending -->\n<!-- module:   -->\n<!-- deltas: 0 -->\n", "", errModule},
		{"module with a pipe", "2026-09-01-a.md", "<!-- review-status: pending -->\n<!-- module: a|b -->\n<!-- deltas: 0 -->\n", "", errModule},
		{"module below line 10", "2026-09-01-a.md",
			"<!-- review-status: pending -->\n<!-- deltas: 0 -->\n\n\n\n\n\n\n\n\n<!-- module: kit -->\n", "", errModule},
		{"no deltas", "2026-09-01-a.md", "<!-- review-status: pending -->\n<!-- module: kit -->\n", "", errDeltas},
		{"non-integer deltas", "2026-09-01-a.md", "<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: many -->\n", "", errDeltas},
		{"no date", "plan.md", ok, "", errDate},
		{"no trailing newline", "2026-09-01-a.md", strings.TrimSuffix(ok, "\n# x\n"),
			"| 2026-09-01-a.md | kit | 2026-09-01 | pending | 2 |", ""},
	}
	for _, c := range cases {
		e, reason, err := ParseHeader(c.file, strings.NewReader(c.text))
		if err != nil {
			t.Errorf("%s: unexpected read error %v", c.name, err)
			continue
		}
		if reason != c.reason {
			t.Errorf("%s: reason = %q, want %q", c.name, reason, c.reason)
			continue
		}
		if c.reason == "" && e.Row != c.row {
			t.Errorf("%s: row = %q, want %q", c.name, e.Row, c.row)
		}
	}
}

func TestRenderSortsByDateThenName(t *testing.T) {
	t.Parallel()
	// Mutation: GRI-order -- sorting by filename only puts APPLY-PLAN (A > 2) after a later-dated retro
	in := []Entry{
		{Date: "2026-09-02", Name: "2026-09-02-zeta.md", Row: "z"},
		{Date: "2026-09-01", Name: "APPLY-PLAN-2026-09-01-plan.md", Row: "p"},
		{Date: "2026-09-01", Name: "2026-09-01-beta.md", Row: "b"},
		{Date: "2026-09-01", Name: "2026-09-01-alpha.md", Row: "a"},
	}
	got := Render(in)
	if !strings.HasPrefix(got, gen.GeneratedLine+"\n") {
		t.Fatalf("Render does not start with the GENERATED marker:\n%s", got)
	}
	if want := "|---|---|---|---|---|\na\nb\np\nz"; !strings.HasSuffix(got, want) {
		t.Errorf("Render rows/tail = %q, want suffix %q", got[len(got)-30:], want)
	}
	if empty := Render(nil); !strings.HasSuffix(empty, "|---|---|---|---|---|") {
		t.Errorf("Render(nil) must end at the table separator with no trailing newline")
	}
}

// TestGoldenRealTree generates the index of the real retros: every committed retro header parses and
// yields exactly one row. retros/INDEX.md itself is not committed, so there is no file to compare with.
func TestGoldenRealTree(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "build-n4-module-kit")
	var diag strings.Builder
	got, code := Generate(root, &diag)
	if code != ExitOK {
		t.Fatalf("real retros have header problems: exit %d: %s", code, diag.String())
	}
	names, err := filepath.Glob(filepath.Join(root, "retros", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	retros := 0
	for _, n := range names {
		if b := filepath.Base(n); b != "INDEX.md" {
			retros++
			if !strings.Contains(got, "\n| "+b+" |") {
				t.Errorf("no INDEX row for %s", b)
			}
		}
	}
	if rows := strings.Count(got, "\n| ") - 1; rows != retros { // minus the table header row
		t.Errorf("generated %d rows for %d retros", rows, retros)
	}
}

func TestScanSkipsIndexDotfilesAndNonFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("INDEX.md", "anything")
	write(".hidden-2026-09-01.md", "anything")
	write("notes.txt", "anything")
	write("2026-09-02-bad.md", "<!-- review-status: pending -->\n")
	write("2026-09-01-ok.md", "<!-- review-status: folded -->\n<!-- module: kit -->\n<!-- deltas: 1 -->\n")
	if err := os.Mkdir(filepath.Join(dir, "2026-09-03-dir.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, problems, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "2026-09-01-ok.md" {
		t.Errorf("entries = %+v, want only 2026-09-01-ok.md", entries)
	}
	if len(problems) != 1 || problems[0].Name != "2026-09-02-bad.md" || problems[0].Reason != errModule || problems[0].IO {
		t.Errorf("problems = %+v, want one module violation for 2026-09-02-bad.md", problems)
	}
}

// The review advisory of #288: a read error must never look like a short header.
func TestParseHeaderReportsReadErrors(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	r := io.MultiReader(strings.NewReader("<!-- review-status: pending -->\n"), iotest.ErrReader(boom))
	if _, _, err := ParseHeader("2026-09-01-a.md", r); !errors.Is(err, boom) {
		t.Errorf("ParseHeader err = %v, want the reader's error", err)
	}
}

func TestScanFailsClosedOnAnUnlistableDir(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	entries, problems, err := Scan(file)
	if err == nil || entries != nil || problems != nil {
		t.Errorf("Scan(file) = %v, %v, %v; want only an error", entries, problems, err)
	}
}

func TestScanReportsAnUnstatableRetroAsIO(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "2026-09-01-a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o444); err != nil { // listable, but no entry can be stat'ed
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	entries, problems, err := Scan(dir)
	if err != nil || len(entries) != 0 || len(problems) != 1 || !problems[0].IO || problems[0].Name != "2026-09-01-a.md" {
		t.Errorf("Scan = %v, %+v, %v; want one IO problem", entries, problems, err)
	}
}
