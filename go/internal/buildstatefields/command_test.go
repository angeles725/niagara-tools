package buildstatefields

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/buildstate"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/retroindex"
)

const hand = "| Retro file | review-status |\n|---|---|\n"

// fixture is a kit root with BUILD-STATE.md, retros/ (files) and retros/INDEX.md (index).
type fixture struct{ root, state, retros, index string }

func newKit(t *testing.T, state, index string, retros map[string]string) fixture {
	t.Helper()
	root := t.TempDir()
	f := fixture{root, filepath.Join(root, "BUILD-STATE.md"), filepath.Join(root, "retros"), filepath.Join(root, "retros", "INDEX.md")}
	mustMkdir(t, f.retros)
	writeFile(t, f.state, state)
	writeFile(t, f.index, index)
	for name, text := range retros {
		writeFile(t, filepath.Join(f.retros, name), text)
	}
	return f
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(string) string { return "" }})
	return code, o.String(), e.String()
}

func (f fixture) sweep() (int, string, string) { return run(f.state, f.retros, f.index) }

// writeState mirrors tests/build-retro-sync.bats write_state.
func writeState(required, pending string) string {
	s := "# BUILD-STATE\n\n<!-- build-state.v1 -->\nschema: build-state.v1\nmodule: DemoPan\nmodule_repo: Cliente/Demo\n" +
		"module_root: Demo/DemoPan\nretro_required: " + required + "\n"
	if pending != "OMIT" {
		s += "retro_pending: " + pending + "\n"
	}
	return s + "last_commit: abc1234\nlast_session: 2026-09-04\n<!-- /build-state.v1 -->\n"
}

func row(file, status string) string { return "| " + file + " | " + status + " |\n" }

func retro(marker string) string { return marker + "\n\n# demo retro\n" }

func TestMainContent(t *testing.T) {
	t.Parallel()
	pendingRetro := map[string]string{"2026-09-04-demo.md": retro("<!-- review-status: pending -->")}
	cases := []struct {
		name         string
		state, index string
		retros       map[string]string
		code         int
		errs         string
	}{
		{"S1: clean state + matching retro + row", writeState("true", "false"), hand + row("2026-09-04-demo.md", "pending"), pendingRetro, ExitOK, ""},
		{"S2: unbalanced marker", "<!-- build-state.v1 -->\nmodule: DemoPan\n", hand, nil, ExitFail, "sweep: unbalanced build-state.v1 markers (1 open / 0 close)\n"},
		{"S3: a retro with no INDEX row", writeState("true", "false"), hand, pendingRetro, ExitFail, "sweep: retro has no INDEX row: 2026-09-04-demo.md\n"},
		{"S4: retro_pending missing", writeState("true", "OMIT"), hand, nil, ExitFail, "sweep: envelope (DemoPan) missing required field: retro_pending\n"},
		{"S5: a row pointing at a missing retro", writeState("false", "false"), hand + row("2026-09-04-ghost.md", "pending"), nil, ExitFail, "sweep: INDEX row points at a missing retro: 2026-09-04-ghost.md\n"},
		{"S6: a bad review-status", writeState("true", "false"), hand + row("2026-09-04-demo.md", "banana"),
			map[string]string{"2026-09-04-demo.md": retro("<!-- review-status: banana -->")}, ExitFail, "sweep: INDEX row for 2026-09-04-demo.md has no valid review-status (pending|folded)\n"},
		{"S8: prose-embedded markers are not envelopes",
			"## How to read\n- one `<!-- build-state.v1 -->`\n  `<!-- /build-state.v1 -->` per module\n\n<!-- build-state.v1 -->\nmodule: DemoPan\nretro_required: false\nretro_pending: false\n<!-- /build-state.v1 -->\n",
			hand, nil, ExitOK, ""},
		{"S9: a multi-line open_issues list",
			"<!-- build-state.v1 -->\nmodule: DemoPan\nretro_required: true\nretro_pending: false\nopen_issues:\n  - DefrostController.java has zero pure tests (HIGH).\n  - suction-2 sensor stuck at 130.5342 psi.\nlast_commit: abc1234\n<!-- /build-state.v1 -->\n",
			hand + row("2026-09-04-demo.md", "pending"), pendingRetro, ExitOK, ""},
		{"M1: marker disagrees with the row", writeState("true", "false"), hand + row("2026-09-04-demo.md", "pending"),
			map[string]string{"2026-09-04-demo.md": retro("<!-- review-status: folded -->")}, ExitFail,
			"sweep: retro marker disagrees with INDEX row: 2026-09-04-demo.md marker='folded' INDEX='pending'\n"},
		{"M2: marker agrees", writeState("true", "false"), hand + row("2026-09-04-demo.md", "pending"), pendingRetro, ExitOK, ""},
		{"M3: a missing marker is tolerated", writeState("true", "false"), hand + row("2026-09-04-demo.md", "folded"),
			map[string]string{"2026-09-04-demo.md": retro("")}, ExitOK, ""},
		{"M4: an out-of-domain marker in the real shape", writeState("true", "false"), hand + row("2026-09-02-demo.md", "folded"),
			map[string]string{"2026-09-02-demo.md": retro("<!-- review-status: fresh . 2026-09-02 -->")}, ExitFail,
			"sweep: retro marker out of domain: 2026-09-02-demo.md has review-status 'fresh' (pending|folded)\n"},
		{"M6: a review-status mention past line 5 is not a marker", writeState("true", "false"), hand + row("2026-09-05-prose.md", "folded"),
			map[string]string{"2026-09-05-prose.md": "# Retro: something\n\n## Context\n\nBody line.\n\nEarlier the review-status: pending marker was discussed here in prose.\n"}, ExitOK, ""},
		{"an unterminated last INDEX row is checked", writeState("true", "false"), hand + "| 2026-09-04-demo.md | draft |", pendingRetro, ExitFail,
			"sweep: INDEX row for 2026-09-04-demo.md has no valid review-status (pending|folded)\n"},
		{"dot files and non-.md files in retros/ need no row", writeState("true", "false"), hand,
			map[string]string{".hidden.md": "", "notes.txt": ""}, ExitOK, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newKit(t, tt.state, tt.index, tt.retros)
			code, out, errs := f.sweep()
			if code != tt.code || out != "" || errs != tt.errs {
				t.Errorf("got (%d, %q, %q)\nwant (%d, \"\", %q)", code, out, errs, tt.code, tt.errs)
			}
		})
	}
}

func TestMainUsageAndEnv(t *testing.T) {
	t.Parallel()
	f := newKit(t, writeState("true", "false"), hand, nil)
	nope := filepath.Join(f.root, "nope")
	cases := []struct {
		name string
		args []string
		errs string
	}{
		{"S7: one argument", []string{f.state}, usageText},
		{"no arguments", nil, usageText},
		{"four arguments", []string{f.state, f.retros, f.index, "x"}, usageText},
		{"no state file", []string{nope, f.retros, f.index}, "sweep: no BUILD-STATE file: " + nope + "\n"},
		{"state is a directory", []string{f.retros, f.retros, f.index}, "sweep: no BUILD-STATE file: " + f.retros + "\n"},
		{"no retros dir", []string{f.state, nope, f.index}, "sweep: no retros dir: " + nope + "\n"},
		{"no INDEX file", []string{f.state, f.retros, nope}, "sweep: no INDEX file: " + nope + "\n"},
	}
	for _, tt := range cases {
		if code, out, errs := run(tt.args...); code != ExitEnv || out != "" || errs != tt.errs {
			t.Errorf("%s: got (%d, %q, %q)", tt.name, code, out, errs)
		}
	}
}

func TestMainUnreadableState(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 file")
	}
	f := newKit(t, writeState("true", "false"), hand, nil)
	if err := os.Chmod(f.state, 0); err != nil {
		t.Fatal(err)
	}
	code, _, errs := f.sweep()
	if code != ExitEnv || !strings.HasPrefix(errs, "sweep: cannot read BUILD-STATE file: "+f.state+": ") {
		t.Errorf("got (%d, %q)", code, errs)
	}
}

// genKit is a kit whose retros/INDEX.md is GENERATED by gen-retro-index from one retro.
func genKit(t *testing.T) fixture {
	t.Helper()
	f := newKit(t, "<!-- build-state.v1 -->\nmodule: kit\nretro_required: true\nretro_pending: true\n<!-- /build-state.v1 -->\n", "",
		map[string]string{"2026-09-01-alpha.md": "<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 1 -->\n# alpha\n"})
	var o, e bytes.Buffer
	if retroindex.Main([]string{f.root}, cli.Stdio{Out: &o, Err: &e}) != 0 {
		t.Fatalf("gen-retro-index: %s", e.String())
	}
	return f
}

func TestMainGeneratedIndex(t *testing.T) {
	t.Parallel()
	f := genKit(t)
	if code, out, errs := f.sweep(); code != ExitOK || out != "" || errs != "" {
		t.Fatalf("GRI-sweep fresh: got (%d, %q, %q)", code, out, errs)
	}
	// Mutation: GRI-sweep -- skipping the freshness call lets a deltas header drift pass
	writeFile(t, filepath.Join(f.retros, "2026-09-01-alpha.md"), "<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 2 -->\n# alpha\n")
	code, out, errs := f.sweep()
	if code != ExitFail || out != "" || errs != "sweep: "+f.index+" is stale vs the retro headers; run toolbelt/gen-retro-index.sh\n" {
		t.Errorf("GRI-sweep stale: got (%d, %q, %q)", code, out, errs)
	}
	writeFile(t, filepath.Join(f.retros, "2026-09-01-alpha.md"), "<!-- review-status: pending -->\n# alpha\n")
	code, _, errs = f.sweep()
	if code != ExitFail || !strings.HasSuffix(errs, "sweep: the retro headers under "+f.retros+" do not generate an INDEX\n") {
		t.Errorf("GRI-sweep malformed header: got (%d, %q)", code, errs)
	}
}

// TestMainGeneratedIndexFromATempDir: the index is not committed, so CI and the pre-push hook pass
// fresh kit-state output from a temp dir; a generated INDEX is compared wherever it lives.
func TestMainGeneratedIndexFromATempDir(t *testing.T) {
	t.Parallel()
	f := genKit(t)
	other := filepath.Join(t.TempDir(), "INDEX.md")
	b, err := os.ReadFile(f.index)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, other, string(b))
	if err := os.Remove(f.index); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run(f.state, f.retros, other); code != ExitOK || out != "" || errs != "" {
		t.Errorf("fresh temp index: got (%d, %q, %q)", code, out, errs)
	}
	writeFile(t, other, string(b)+"| 2026-09-02-ghost.md | kit | 2026-09-02 | pending | 1 |\n")
	if code, _, errs := run(f.state, f.retros, other); code != ExitFail || errs != "sweep: INDEX row points at a missing retro: 2026-09-02-ghost.md\n" {
		t.Errorf("stale temp index: got (%d, %q)", code, errs)
	}
}

// ledgerKit adds ledger.d/ beside BUILD-STATE.md with a kit block generated from it.
func ledgerKit(t *testing.T) fixture {
	t.Helper()
	f := newKit(t, "<!-- build-state.v1 -->\nmodule: kit\nlast_change: x\nopen_issues:\nretro_required: true\nretro_pending: true\nlast_session: 2026-09-04 · demo\n<!-- /build-state.v1 -->\n", hand, nil)
	mustMkdir(t, filepath.Join(f.root, "ledger.d"))
	writeFile(t, filepath.Join(f.root, "ledger.d", "0000-archive-pre-2026-10-23.md"), "<!-- ledger: kit -->\n<!-- date: 2026-09-04 -->\nlog: 2026-09-04 · archived line\n")
	writeFile(t, filepath.Join(f.root, "ledger.d", "2026-09-05-demo.md"), "<!-- ledger: kit -->\n<!-- date: 2026-09-05 -->\nlog: 2026-09-05 · demo change\nopen_issue: OWED demo half\n")
	var o, e bytes.Buffer
	if buildstate.Main([]string{f.root}, cli.Stdio{Out: &o, Err: &e}) != 0 {
		t.Fatalf("gen-build-state: %s", e.String())
	}
	return f
}

func TestMainLedger(t *testing.T) {
	t.Parallel()
	f := ledgerKit(t)
	if code, out, errs := f.sweep(); code != ExitOK || out != "" || errs != "" {
		t.Fatalf("GBS-sweep fresh: got (%d, %q, %q)", code, out, errs)
	}
	// Mutation: GBS-sweep -- skipping this call lets a fragment land without its regenerated BUILD-STATE.md
	writeFile(t, filepath.Join(f.root, "ledger.d", "2026-09-06-late.md"), "<!-- ledger: kit -->\n<!-- date: 2026-09-06 -->\nlog: 2026-09-06 · late\n")
	code, out, errs := f.sweep()
	if code != ExitFail || out != "" ||
		!strings.HasPrefix(errs, "gen-build-state: "+f.root+"/BUILD-STATE.md is stale vs ledger.d/; run gen-build-state.sh and commit the result\n") ||
		!strings.HasSuffix(errs, "sweep: BUILD-STATE.md is stale vs ledger.d/ (or a fragment is malformed); run toolbelt/gen-build-state.sh\n") {
		t.Errorf("GBS-sweep stale: got (%d, %q, %q)", code, out, errs)
	}

	g := ledgerKit(t)
	renamed := filepath.Join(g.root, "STATE.md")
	b, err := os.ReadFile(g.state)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, renamed, string(b))
	if code, _, errs := run(renamed, g.retros, g.index); code != ExitFail || errs != "sweep: a ledger.d/ kit needs the state file named BUILD-STATE.md: "+renamed+"\n" {
		t.Errorf("renamed state: got (%d, %q)", code, errs)
	}
}

// TestMainRealKit is M5: the sweep on the real kit tree exits 0, in the CI argument shape.
func TestMainRealKit(t *testing.T) {
	t.Parallel()
	k := filepath.Join("..", "..", "..", "build-n4-module-kit")
	code, out, errs := run(filepath.Join(k, "BUILD-STATE.md"), filepath.Join(k, "retros"), filepath.Join(k, "retros", "INDEX.md"))
	if code != ExitOK || out != "" || errs != "" {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}

// addRow mirrors tests/retro-debt.bats add_row.
func addRow(t *testing.T, f fixture, stem, status string) {
	t.Helper()
	writeFile(t, filepath.Join(f.retros, stem+".md"), "<!-- review-status: "+status+" -->\n\n# retro\n")
	b, err := os.ReadFile(f.index)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, f.index, string(b)+row(stem+".md", status))
}

func TestMainAge(t *testing.T) {
	t.Parallel()
	report := func(total, esc int, oldest string) string {
		return "total_pending=" + strconv.Itoa(total) + "\nescalated_count=" + strconv.Itoa(esc) + "\noldest_age=" + oldest + "\n"
	}
	cases := []struct {
		name  string
		rows  [][2]string
		extra string
		want  string
	}{
		{"D1: no rows", nil, "", report(0, 0, "N/A")},
		{"D2: a folded row never ages", [][2]string{{"2026-01-01-old-folded", "folded"}}, "", report(0, 0, "N/A")},
		{"D3: same day is age 0", [][2]string{{"2026-09-05-today", "pending"}}, "", report(1, 0, "0")},
		{"D4: exactly 30 days is not escalated", [][2]string{{"2026-08-06-thirty", "pending"}}, "", report(1, 0, "30")},
		{"D5: 31 days is escalated", [][2]string{{"2026-08-05-thirtyone", "pending"}}, "", report(1, 1, "31")},
		{"D6: mixed", [][2]string{{"2026-08-05-esc", "pending"}, {"2020-01-01-ancient", "folded"}, {"2026-09-01-recent", "pending"}}, "", report(2, 1, "31")},
		{"D7: a tracked 0-delta row is not debt", [][2]string{{"2026-08-05-esc", "pending"}},
			"| 2026-08-01-proposal.md | pending | 0 | <!-- disposition: tracked-by-issue #1 -->\n", report(1, 1, "31")},
		{"D8: a disposition on a row with deltas still ages", nil,
			"| 2026-08-01-real.md | pending | 3 | <!-- disposition: tracked-by-issue #1 -->\n", report(1, 1, "35")},
		{"D9: a disposition outside the allowed set still ages", nil,
			"| 2026-08-01-parked.md | pending | 0 | <!-- disposition: later -->\n| 2026-08-02-done.md | pending | 0 | <!-- disposition: worklist-complete — all applied -->\n", report(1, 1, "35")},
		{"a future row ages negative and never raises oldest_age", nil, "| 2026-10-01-future.md | pending |\n", report(1, 0, "-1")},
		{"an impossible date counts but does not age", nil, "| 2026-02-30-bad.md | pending |\n| 2026-08-05-x.md | pending |\n", report(2, 1, "31")},
		{"the first .md name of the row carries the date", nil, "| notes.md 2026-03-03-z.md | pending |\n", report(1, 0, "-1")},
		{"an unterminated last row ages", nil, "| 2026-08-05-x.md | pending |", report(1, 1, "31")},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newKit(t, "", hand, nil)
			for _, r := range tt.rows {
				addRow(t, f, r[0], r[1])
			}
			b, _ := os.ReadFile(f.index)
			writeFile(t, f.index, string(b)+tt.extra)
			code, out, errs := run("--age", "--today", "2026-09-05", f.retros, f.index)
			if code != ExitOK || out != tt.want || errs != "" {
				t.Errorf("got (%d, %q, %q), want %q", code, out, errs, tt.want)
			}
		})
	}
}

func TestMainAgeArgs(t *testing.T) {
	t.Parallel()
	f := newKit(t, "", hand, nil)
	addRow(t, f, "2026-08-05-x", "pending")
	nope := filepath.Join(f.root, "nope")
	cases := []struct {
		name      string
		args      []string
		code      int
		out, errs string
	}{
		{"--max-age widens the window", []string{"--age", "--max-age", "40", "--today", "2026-09-05", f.retros, f.index}, ExitOK, "total_pending=1\nescalated_count=0\noldest_age=31\n", ""},
		{"--max-age 0", []string{"--age", "--today", "2026-09-05", "--max-age", "0", f.retros, f.index}, ExitOK, "total_pending=1\nescalated_count=1\noldest_age=31\n", ""},
		{"the last --today wins", []string{"--age", "--today", "2026-01-01", "--today", "2026-09-05", f.retros, f.index}, ExitOK, "total_pending=1\nescalated_count=1\noldest_age=31\n", ""},
		{"-- ends the flags", []string{"--age", "--today", "2026-09-05", "--", f.retros, f.index}, ExitOK, "total_pending=1\nescalated_count=1\noldest_age=31\n", ""},
		{"no --today", []string{"--age", f.retros, f.index}, ExitEnv, "", "sweep-build-state: --today <YYYY-MM-DD> is required\n"},
		{"--today without a value", []string{"--age", "--today"}, ExitEnv, "", ageUsageText},
		{"--max-age without a value", []string{"--age", "--today", "2026-09-05", "--max-age"}, ExitEnv, "", ageUsageText},
		{"an unknown flag", []string{"--age", "--today", "2026-09-05", "--strict", f.retros, f.index}, ExitEnv, "", ageUsageText},
		{"one positional", []string{"--age", "--today", "2026-09-05", f.retros}, ExitEnv, "", ageUsageText},
		{"three positionals", []string{"--age", "--today", "2026-09-05", f.retros, f.index, "x"}, ExitEnv, "", ageUsageText},
		{"no retros dir", []string{"--age", "--today", "2026-09-05", nope, f.index}, ExitEnv, "", "sweep-build-state: no retros dir: " + nope + "\n"},
		{"no INDEX file", []string{"--age", "--today", "2026-09-05", f.retros, nope}, ExitEnv, "", "sweep-build-state: no INDEX file: " + nope + "\n"},
		{"--today is not a date", []string{"--age", "--today", "garbage", f.retros, f.index}, ExitEnv, "", "sweep-build-state: --today is not a YYYY-MM-DD date: garbage\n"},
		{"--max-age is not a number", []string{"--age", "--today", "2026-09-05", "--max-age", "x", f.retros, f.index}, ExitEnv, "", "sweep-build-state: --max-age is not a whole number: x\n"},
	}
	for _, tt := range cases {
		if code, out, errs := run(tt.args...); code != tt.code || out != tt.out || errs != tt.errs {
			t.Errorf("%s: got (%d, %q, %q)\nwant (%d, %q, %q)", tt.name, code, out, errs, tt.code, tt.out, tt.errs)
		}
	}
}
