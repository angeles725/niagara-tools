<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-23 · kit · go-e2-tiny-lints

**Session**: niagara-research quality round 8, work unit E2, branch `r8/e2-tiny-lints` from origin/main 63c5b3b
**Delta count**: 2

## What happened
Seven small line-matching lints moved onto the Go lint engine, one rule package each:
`lint-no-system-out` (nosystemout), `lint-se-display` (sedisplay), `lint-bql-string-concat` (bqlconcat),
`lint-clock-zero-floor` (clockzerofloor), all four `awk_lint_each` lints, and `lint-subscribe-without-unsubscribe`
(subscribeleak), `lint-wb-file-chooser` (wbfilechooser), `lint-no-md5-credential-digest` (nomd5credential), all
three `awk_lint_strip_run` lints. Every rule sets `ContinueOnWalkError` (both helpers listed with `awk_lint_list`,
which scans the partial list) and the scripts are wrappers that exec `n4lint <rule>`. The shared code is one small
package, `go/internal/lint/rules/linescan`: the row texts (`Source`, `Trimmed`) and `Code`, the comment- and
literal-blanked view. Each port landed byte-identical first; then one fix commit per lint removed the bug all seven
shared: the line was cut at its first `//` without string awareness (truth audit T3).

## Evidence
- RED: 140 golden cases recorded from the old scripts (every bats run plus 14 edge trees per lint) failed 65 of 140
  against a stub `Check`; the others expect no rows or test the argument contract
  `[ev: go/internal/lint/rules/nosystemout/nosystemout_test.go]`
- Byte identity, old script vs wrapper, stdout + stderr + exit: same on tests/fixtures, Cliente, Honeywell,
  panccadia-leon, Leon-Guanjuato, a 195-file sample of decompiled Niagara sources (538 / 10 / 1 / 1 rows), no args,
  `-h`, a missing root, a file root, two roots, `--strict`, `--`, a bad flag, a symlinked root and a backslash path
- Declared differences of the ports: an unreadable file prints the engine text `cannot scan <f>: awk: fatal: cannot
  open file ...` (was `cannot scan <f>: Permission denied` from awk_lint_each, `sed: can't read` from
  awk_lint_strip_run; the four `*-unreadable` bats expectations follow); a path holding a newline is scanned (the bash
  file list split it into two missing paths); bash's `ignored null byte in input` warning (a NUL in Honeywell
  ChiJsonUtilTest.java) is gone
- Timing, old → Go, one run each: Cliente 0.45-0.94 s → 0.05-0.16 s per lint, panccadia-leon 0.19-0.32 s → 0.03-0.05 s
- Truth fixes, each RED first (a hand-written `truth-*.txt` golden and a bats test with a `# Mutation:` pin, both
  failing on the port and on the old script): a match in `/* */`, Javadoc or a string is gone, a token after a `//`
  inside a string is found, a commented `unsubscribe` / `BComponentChooser` / credential word no longer exempts or
  arms a file, and lint-no-system-out matches the statement across lines with `System` as a whole word
- Real-tree rows after the fixes: unchanged on Cliente, Honeywell, panccadia-leon, Leon-Guanjuato and tests/fixtures;
  on the decompiled sample lint-no-system-out gains 17 split calls (`System.out\n  .println(`, lonworks-rt, nre-ext,
  niagarad-ext) and drops 3 code-generator strings (program-wb ComponentWriter `wl("... System.out.println ...")`)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Let a spec choose the unreadable-file text.** The engine prints awk_lint_run's `awk: fatal: cannot open file` for every rule, so a port of an `awk_lint_each` or `awk_lint_strip_run` lint changes its stderr and its bats expectation. Why: four bats expectations of this unit changed only for that text, and a Go rule printing `awk: fatal` misleads. Reviewer check: a port that edits an exact `cannot scan` expectation names the delta. | `go/internal/lint/run.go` § Run | `[ev: tests/lint-no-system-out.bats]` |
| Δ2 | **List files NUL-separated in `scan_files`.** The newline-separated list splits a path holding a newline into two missing paths, a false env error that also skips the file, in every remaining bash lint. Why: all seven bash lints of this unit printed two `No such file` lines for one such file. Reviewer check: a new `while read -r f` over a file list is a finding. | `toolbelt/lib/scan-files.sh` | `[ev: build-n4-module-kit/toolbelt/lib/awk-lint.sh]` |

## Lessons
- Port byte-identical first, then fix: the truth fixes changed recorded goldens only in the `edge-comment-literal`
  case of each lint, which makes every behaviour change reviewable on its own.
- A token that lives in a literal (`bql:`, `"MD5"`, a `"targetType"` facet key, a `"password"` slot name) is matched in
  the comment-blanked text (`Stripped`), a code token in `linescan.Code`; blanking literals for every token would lose
  real hits.
- Real client trees rarely exercise comment or string shapes; a decompiled-source sample did (17 + 3 rows).

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-go-e2-tiny-lints.md | kit | 2026-10-23 | pending | 2 |`
