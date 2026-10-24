<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-24 · kit · go-lint-manual-labels

**Session**: niagara-research quality round 10 (Ayudante2, Go lint port), branch `r10/manual-labels-go` from origin/main 2e47c68
**Delta count**: 4

## What happened
`toolbelt/lint-manual-labels.sh` (a python3 heredoc) is now the rule `go/internal/lint/rules/manuallabels` and a
thin wrapper that execs `n4lint lint-manual-labels`; python3 is no longer needed. Its contract (`[--strict]
<manual> <ui-dir>...`, one haystack over every ui-dir, one SUMMARY row) matched no engine shape, so the engine
gained `Mode FileFirst`, `Spec.Joined` and severity `Summary` in their own commit with tests. Python's text
semantics live in a new shared package `go/internal/htmltext` (an `errors="replace"` read, `html.unescape`, the
three markup-dropping passes, and `str` facts and folds generated from Python 3.15's `unicodedata`, since Go has no
NFKD, no full case folding and another Unicode version). The port went byte-identical first, then four truth fixes,
each RED first.

## Evidence
- 52 goldens recorded from origin/main's script through a recording wrapper (the 12 bats runs and 40 edge trees:
  each quote style, short-quote mis-pairing, label lengths, entities, nested tags, script/style/comments, wrapped
  labels, code blocks, case and whitespace, CRLF, lone CR, no final newline, BOM, invalid UTF-8, surrogate escapes,
  extension case, pruning, several ui-dirs, the argument contract); RED: 41/52 failed against a stub `CheckTree`
  `[ev: go/internal/lint/rules/manuallabels/testdata]`
- Byte identity old vs new (stdout, stderr, exit, `cmp`): both real manuals (Dashboard `manual-configuracion.html`
  147 checked / 38 missing; Leon-Guanjuato `MANUAL-modulos-panccadia.md` 7 / 2), `--strict` over two ui-dirs, and
  150 sampled Cliente and kit `*.html`/`*.md` files used as manuals: 150 same, 0 diff. Timing: html manual
  0.90 s → 0.22 s; md manual over Leon-Guanjuato (98 MB) 2.4-2.9 s → 0.92 s `[ev: toolbelt/lint-manual-labels.sh]`
- Declared port differences: `*-worktrees` directories pruned (edge-worktrees), symlinks below a ui-dir not read,
  an unreadable directory or file is exit 3 naming it (python skipped the directory or crashed), script/style tag
  names fold in ASCII only
- Truth fixes (bats ML11-ML14 RED on the port, each mutation observed): a ui-dir with no UI file is the K20 ERROR
  row (was every label missing, exit 0); a label the UI builds from a literal stem ending in a space, "(" or ":"
  plus one word is found (Dashboard manual 38 → 29 rows, the 9 removed rows each verified as `"<stem>" + value` in
  the UI source); quoted strings in code (language-tagged fences, code spans, `<code>`) are not labels while a bare
  fence or `<pre>` diagram is (real manuals unchanged); a label wrapped onto the next non-blank line is one label
  (real manuals unchanged; 17 of the 150 sample files change rows, mostly wrapped phrases now checked)
  `[ev: go/internal/lint/rules/manuallabels/manuallabels.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A stale label kept in a UI comment hides the drift.** The haystack reads JS/Java comments and HTML comments, so a renamed label left in `// "Corte"` or commented-out code counts as found (edge-ui_units: "Nueve", "Once"). Blank comments first: a `jssrc` code view (comments blanked, literals kept, line numbers kept) for `.js/.mjs/.cjs`, `javasrc.Strip` for `.java`, `<!-- -->` for markup; real manuals were unchanged in a prototype, so it is a guard, not a row change. | `go/internal/jssrc` + `rules/manuallabels` | `[ev: retro go-lint-manual-labels]` |
| Δ2 | **A `.properties`/`.lexicon` value written `key:value`, `key value` or over a continuation line is not a unit** (edge-ui_units: "Dieciseis"), a false WARN. Read those files with `props.Read` (java.util.Properties rules) instead of the `key=value` regex. | `rules/manuallabels` | `[ev: go/internal/props/props.go]` |
| Δ3 | **A manual placeholder for a literal family still WARNs**: "Paro N" / "Rearranque N" against the literals "Paro 1".."Paro 3", "Paro/Rearranque 1, 2, 3", and a two-level template ("Protección anti-hielo activa: Evap M"). Decide whether a single-capital placeholder may stand for a digit suffix before matching it; each is a human-read advisory row today. | `toolbelt/lint-manual-labels.sh` header | `[ev: retro go-lint-manual-labels]` |
| Δ4 | **Python `str` facts belong in their own package once a second lint needs them.** `htmltext/unicode.go` (NFKD without marks, casefold, isspace/isalnum/isdecimal, generated from `unicodedata`) is not HTML-specific; move it to e.g. `internal/pystr` with its generator test when a second consumer appears. | `go/README.md` § Layout | `[ev: go/internal/htmltext/unicode.go]` |

## Lessons
- A Python regex port is a set of leftmost, non-overlapping scans with its backtracking outcomes written down
  (a failed quote resumes at the next code point, `\s*` crosses newlines, `.` stops at `\n` only); each scanner got
  a table test with values taken from the helper itself before the goldens ran.
- Text an agent writes through a tool can arrive with `\uXXXX` escapes already decoded: two test literals meant to
  hold a backslash-u held the character instead and still passed. Check escape-sensitive fixtures with `od -c`.
- A truth fix measured only on the real manuals can hide a regression: blanking bare fences dropped a real label
  in a diagram ("Enfriando", 7 → 6 checked), visible only in the SUMMARY count.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-b`) — Δ1, Δ2 and Δ3 not folded, needs code (comment blanking, `props.Read` for lexicons, a placeholder-family decision), owed in `METHODOLOGY.md` § Kit maintenance (round-10 lint follow-ups) and `BUILD-STATE.md` open_issues; Δ4 cross-cutting Go-port lesson, cited by the lints-a fold. INDEX row generated from the header: `| 2026-10-24-go-lint-manual-labels.md | kit | 2026-10-24 | folded | 4 |`
