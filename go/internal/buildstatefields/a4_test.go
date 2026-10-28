package buildstatefields

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/buildstate"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// regen regenerates the fixture's BUILD-STATE.md after a fragment edit, so only the check under test can fail.
func regen(t *testing.T, f fixture) {
	t.Helper()
	var o, e bytes.Buffer
	if buildstate.Main([]string{f.root}, cli.Stdio{Out: &o, Err: &e}) != 0 {
		t.Fatalf("gen-build-state: %s", e.String())
	}
}

// TestMainLedgerEvidenceTags is audit A4: an [ev: tag left open in a ledger fragment (no closing ] before
// the next tag or the end of the text) credits a fold it does not name, so the sweep FAILs it by file:line.
func TestMainLedgerEvidenceTags(t *testing.T) {
	t.Parallel()
	const head = "<!-- ledger: kit -->\n<!-- date: 2026-09-07 -->\n"
	cases := []struct {
		name, log, errs string
		code            int
	}{
		{"a closed tag passes", "log: 2026-09-07 · x [ev: retro demo Δ1]\n", "", ExitOK},
		{"a backticked literal opener is code, not a tag", "log: 2026-09-07 · the `[ev:` opener [ev: retro demo Δ1]\n", "", ExitOK},
		// Mutation: LT1-unterminated -- skipping the ledger tag check lets an open [ev: tag through
		{"an unterminated tag FAILs", "log: 2026-09-07 · x [ev: retro demo Δ1\n",
			"sweep: ledger.d/2026-09-07-tag.md:3: unbalanced [ev: tag (no closing ] before the next tag or the end of the text): [ev: retro demo Δ1\n", ExitFail},
		// Mutation: LT2-nested -- reading only unterminated tags misses an opener swallowed by the previous tag
		{"a tag opened inside an open tag FAILs", "log: 2026-09-07 · x [ev: retro demo Δ1 [ev: retro other Δ2]\n",
			"sweep: ledger.d/2026-09-07-tag.md:3: unbalanced [ev: tag (no closing ] before the next tag or the end of the text): [ev: retro demo Δ1 [ev: retro other Δ2\n", ExitFail},
		// Mutation: LT3-opener -- a fixed "[ev:" label misnames an open [deferred: tag
		{"an open_issue line is read too (deferred opener named)", "log: 2026-09-07 · x\nopen_issue: owed [deferred: retro demo Δ2\n",
			"sweep: ledger.d/2026-09-07-tag.md:4: unbalanced [deferred: tag (no closing ] before the next tag or the end of the text): [deferred: retro demo Δ2\n", ExitFail},
		{"a tag opened inside an open deferred tag names the deferred opener", "log: 2026-09-07 · x [deferred: retro demo Δ1 [ev: retro other Δ2]\n",
			"sweep: ledger.d/2026-09-07-tag.md:3: unbalanced [deferred: tag (no closing ] before the next tag or the end of the text): [deferred: retro demo Δ1 [ev: retro other Δ2\n", ExitFail},
		// Mutation: LT4-backtick -- a plain substring test FAILs a tag whose text quotes the opener as code
		{"a backticked opener inside a closed tag is code", "log: 2026-09-07 · x [ev: retro demo Δ1 `[ev:` literal]\n", "", ExitOK},
		{"a backticked deferred opener inside a closed tag is code", "log: 2026-09-07 · x [ev: retro demo Δ1 `[deferred:`]\n", "", ExitOK},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := ledgerKit(t)
			writeFile(t, filepath.Join(f.root, "ledger.d", "2026-09-07-tag.md"), head+tt.log)
			regen(t, f)
			if code, out, errs := f.sweep(); code != tt.code || out != "" || errs != tt.errs {
				t.Errorf("got (%d, %q, %q)\nwant (%d, %q)", code, out, errs, tt.code, tt.errs)
			}
		})
	}
}

// TestMainSourceOfTruthWarn is audit A4: every module envelope declares module_root and source_of_truth
// (BUILD-STATE.src.md § How to read). A missing or empty one is a WARN on stderr, never a FAIL; `unknown`
// is an honest value; the kit's own envelope has no module tree and is exempt.
func TestMainSourceOfTruthWarn(t *testing.T) {
	t.Parallel()
	full := writeState("true", "false")
	cases := []struct{ name, state, errs string }{
		{"both declared: no WARN", full, ""},
		{"unknown is an honest value", strings.Replace(full, "Cliente/Demo@main@abc1234", "unknown", 1), ""},
		// Mutation: SOT1-warn -- dropping the field WARN lets an envelope lose its source_of_truth silently
		{"no source_of_truth", strings.Replace(full, "source_of_truth: Cliente/Demo@main@abc1234\n", "", 1),
			"sweep: WARN envelope (DemoPan) has no source_of_truth (required for every module: <path>@<branch>@<commit> or unknown)\n"},
		{"an empty module_root", strings.Replace(full, "module_root: Demo/DemoPan", "module_root:   # TODO", 1),
			"sweep: WARN envelope (DemoPan) has no module_root (required for every module: the module's absolute root)\n"},
		{"the kit envelope is exempt", strings.Replace(strings.Replace(full, "module: DemoPan", "module: kit", 1),
			"module_root: Demo/DemoPan\nsource_of_truth: Cliente/Demo@main@abc1234\n", "", 1), ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newKit(t, tt.state, hand+row("2026-09-04-demo.md", "pending"), map[string]string{"2026-09-04-demo.md": retro("<!-- review-status: pending -->")})
			if code, out, errs := f.sweep(); code != ExitOK || out != "" || errs != tt.errs {
				t.Errorf("got (%d, %q, %q), want errs %q", code, out, errs, tt.errs)
			}
		})
	}
}

// TestMainAgeStrict is audit A4: --age --strict prints the same report and exits 1 when a pending retro is
// older than --max-age days (escalated_count > 0), 0 otherwise.
func TestMainAgeStrict(t *testing.T) {
	t.Parallel()
	f := newKit(t, "", hand, nil)
	addRow(t, f, "2026-09-01-recent", "pending")
	code, out, errs := run("--age", "--strict", "--today", "2026-09-05", f.retros, f.index)
	if code != ExitOK || out != "total_pending=1\nescalated_count=0\noldest_age=4\n" || errs != "" {
		t.Errorf("strict, nothing escalated: got (%d, %q, %q)", code, out, errs)
	}
	addRow(t, f, "2026-08-05-old", "pending")
	// Mutation: AGE-strict -- ignoring --strict keeps exit 0 over escalated retro debt
	code, out, errs = run("--age", "--today", "2026-09-05", "--strict", f.retros, f.index)
	if code != ExitFail || out != "total_pending=2\nescalated_count=1\noldest_age=31\n" ||
		errs != "sweep-build-state: --strict: 1 pending retro(s) older than 30 days\n" {
		t.Errorf("strict, escalated: got (%d, %q, %q)", code, out, errs)
	}
	if code, _, _ := run("--age", "--today", "2026-09-05", f.retros, f.index); code != ExitOK {
		t.Errorf("non-strict over the same debt: got exit %d, want 0", code)
	}
}

// TestLedgerTagsEnv: a ledger.d/ the sweep cannot list, or a fragment it cannot read, is an environment error
// (ExitEnv) naming the path, never a clean pass (review R3-ledgertags-coverage of b9709db0).
func TestLedgerTagsEnv(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 file")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "ledger.d")
	mustMkdir(t, dir)
	frag := filepath.Join(dir, "2026-09-07-tag.md")
	writeFile(t, frag, "log: x\n")
	if err := os.Chmod(frag, 0); err != nil {
		t.Fatal(err)
	}
	if msg, code := ledgerTags(root); code != ExitEnv || !strings.HasPrefix(msg, "cannot read ledger.d/2026-09-07-tag.md: ") {
		t.Errorf("unreadable fragment: (%q, %d)", msg, code)
	}
	if err := os.Chmod(dir, 0o300); err != nil { // searchable, not listable
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755); _ = os.Chmod(frag, 0o644) })
	if msg, code := ledgerTags(root); code != ExitEnv || !strings.HasPrefix(msg, "cannot list "+dir+": ") {
		t.Errorf("unlistable ledger.d: (%q, %d)", msg, code)
	}
}
