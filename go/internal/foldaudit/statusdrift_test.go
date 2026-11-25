package foldaudit

import "testing"

const driftIndex = "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n"

// TestMainFoldedHeaderWithPendingBodyStatus: a retro whose line-1 marker says folded but whose body
// `**Status**: PENDING` line was never flipped is reported at its line, counted, and fails --strict.
func TestMainFoldedHeaderWithPendingBodyStatus(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"retros/INDEX.md": driftIndex + "| 2026-10-22-drifted-status.md | kit | 2026-10-22 | folded | 0 |\n",
		"retros/2026-10-22-drifted-status.md": "<!-- review-status: folded -->\n<!-- module: kit -->\n<!-- deltas: 0 -->\n# t\n\n---\n" +
			"**Status**: PENDING — INDEX row appended: `| x | kit | 2026-10-22 | pending | 0 |`\n",
		"ledger.d/a.md": "<!-- ledger: kit -->\nlog: [ev: retro drifted-status]\n",
	})
	want := "fold-audit: WARN 2026-10-22-drifted-status.md:7 header says folded but body Status says PENDING: make it `**Status**: FOLDED …`\n" +
		"fold-audit: 1 folded, 1 cited, 0 uncited\n" +
		"fold-audit: status drift: 1 retros (header folded, body Status PENDING)\n"
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{root + "/retros/INDEX.md", root}, ExitOK},
		{[]string{"--strict", root + "/retros/INDEX.md", root}, ExitStrict},
	} {
		code, out, errs := run("", c.args...)
		if code != c.code || out != want || errs != "" {
			t.Errorf("%q: got (%d, %q, %q), want (%d, %q, \"\")", c.args, code, out, errs, c.code, want)
		}
	}
}

// TestMainStatusAgreesOrNotAudited: nothing is reported when the body says FOLDED, when the retro is still
// pending (header and body agree), when the INDEX row is folded but the retro header is pending, or when the
// retro file is absent; the audit output is then exactly what it was before the check existed.
func TestMainStatusAgreesOrNotAudited(t *testing.T) {
	t.Parallel()
	root := kit(t, map[string]string{
		"retros/INDEX.md": driftIndex +
			"| 2026-10-22-agrees-folded.md | kit | 2026-10-22 | folded | 0 |\n" +
			"| 2026-10-22-still-pending.md | kit | 2026-10-22 | pending | 0 |\n" +
			"| 2026-10-22-stale-index.md | kit | 2026-10-22 | folded | 0 |\n" +
			"| 2026-10-22-no-file-here.md | kit | 2026-10-22 | folded | 0 |\n",
		"retros/2026-10-22-agrees-folded.md": "<!-- review-status: folded -->\n# t\n**Status**: FOLDED 2026-10-22 (branch `x`)\n",
		"retros/2026-10-22-still-pending.md": "<!-- review-status: pending -->\n# t\n**Status**: PENDING — proposed\n",
		"retros/2026-10-22-stale-index.md":   "<!-- review-status: pending -->\n# t\n**Status**: PENDING — proposed\n",
		"ledger.d/a.md":                      "<!-- ledger: kit -->\nlog: [ev: retro agrees-folded] [ev: retro stale-index] [ev: retro no-file-here]\n",
	})
	want := "fold-audit: 3 folded, 3 cited, 0 uncited\n"
	code, out, errs := run("", "--strict", root+"/retros/INDEX.md", root)
	if code != ExitOK || out != want || errs != "" {
		t.Errorf("got (%d, %q, %q), want (0, %q, \"\")", code, out, errs, want)
	}
}
