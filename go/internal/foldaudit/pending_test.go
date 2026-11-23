package foldaudit

import "testing"

// TestMainPendingRowWithEveryDeltaCited (retro build-kit-bench-session Δ19): a pending retro dated since the cutoff
// whose every delta is cited as landed is reported, and fails --strict, so a fold that forgot to flip it is seen.
// A pending retro with a delta still uncited, and one before the cutoff, stay silent.
func TestMainPendingRowWithEveryDeltaCited(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"retros/INDEX.md": "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n" +
			"| 2026-10-22-done.md | kit | 2026-10-22 | pending | 2 |\n" +
			"| 2026-10-22-open.md | kit | 2026-10-22 | pending | 2 |\n" +
			"| 2026-09-01-old.md | kit | 2026-09-01 | pending | 1 |\n" +
			"| 2026-10-23-none.md | kit | 2026-10-23 | pending | 0 |\n",
		"ledger.d/2026-10-22-x.md": "<!-- ledger: kit -->\nlog: 2026-10-22 · [ev: retro done Δ1] [ev: retro done Δ2] [ev: retro open Δ1]\n",
	})
	want := "fold-audit: WARN 2026-10-22-done.md pending, but all 2 deltas are cited: set `<!-- review-status: folded -->` in the retro\n" +
		"fold-audit: 0 folded, 0 cited, 0 uncited\n" +
		"fold-audit: deltas: 0 checked, 0 not cited (folded rows since 2026-09-24)\n" +
		"fold-audit: pending: 1 fully cited\n"
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
	// Without --deltas-since the pending rows are not read, as before.
	if code, out, _ := run("", "--strict", root+"/retros/INDEX.md", root); code != ExitOK || out != "fold-audit: 0 folded, 0 cited, 0 uncited\n" {
		t.Errorf("no cutoff: got (%d, %q)", code, out)
	}
}
