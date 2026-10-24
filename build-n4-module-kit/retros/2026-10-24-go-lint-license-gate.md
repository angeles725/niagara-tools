<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · go-lint-license-gate

**Session**: niagara-research quality round 10, branch `r10/license-gate-go` from origin/main 89390e5
**Delta count**: 2

## What happened
`toolbelt/lint-license-isoperational-gate.sh` (one awk run over every file through `awk_lint_run`, `mb_strip` +
`mb_parse` from lib/method-boundary.sh) is now the per-file rule `go/internal/lint/rules/licensegate` and a thin
wrapper that execs `n4lint lint-license-isoperational-gate`. The port went byte-identical first (javasrc `Stripped`
and `Methods` for mb_strip/mb_parse, the awk `block_body` kept literal-unaware, gawk regexes with `Longest`,
`ContinueOnWalkError` for the awk_lint_run contract, the K20 row as `EmptyError`), then one truth fix with RED
first. The header's mention of lint-status-parity is an overlap note only: the scripts share no helper, row relay
or argument contract, so nothing beyond the text was kept.

## Evidence
- Goldens recorded from the original bash through a recording wrapper (bats suite copy with `LIG` pointed at it,
  plus an edge driver): 13 bats runs on readable trees and 26 edge trees (argument shapes, CRLF, no final newline,
  walk order and pruning, multi-line and abstract declarations, callback names, gate shapes, body shapes, text
  block, unbalanced braces, non-ASCII names, super calls). An empty directory cannot be held by the golden format:
  `TestEmptyDir`. LIG-awkfail and LIG-finderr are `golden.Unreadable`. RED: 21/39 cases and `TestUnreadable` failed
  against a stub `Check` `[ev: go/internal/lint/rules/licensegate/licensegate_test.go]`
- Byte identity old vs new (stdout, stderr, exit), with and without `--strict`: Cliente (0 rows), its panccadia-leon
  and Honeywell groups (0 rows), sources/decompiled (31 rows, exit 1 under `--strict`). Timing, cold then warm:
  Cliente 7.8 s / 3.3 s → 0.11 s / 0.07 s, decompiled 30.2 s / 30.6 s → 3.4 s / 0.48 s `[ev: toolbelt/lint-license-isoperational-gate.sh]`
- Truth fix (RED: three hand-written `truth-*` cases and bats LIG-literal): the rule reads `linescan.Code` (literal
  contents removed) instead of mb_strip's text. Before, a quoted `"isOperational()"` gated a callback, a quoted
  `getLicenseFeature()` declaration licensed a class, and a `;` inside a string cut the first statement before its
  gate. Two recorded edge cases changed as declared (edge-gates gains `doE`, edge-lic-comment loses `BT.doA`); real
  trees unchanged (Cliente 0 → 0, decompiled 31 → 31) `[ev: go/internal/lint/rules/licensegate/licensegate.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A truth case must flip the outcome, not just exercise the bug.** A brace inside a literal truncates the awk's body, but in every callback that holds a literal the body already acts, so a brace case passed on the buggy code; the discriminating case was a `;` inside a literal in the gate's own statement. Reviewer check: each `truth-*` golden fails on the code before its fix (watch it fail), never only on paper. | `go/README.md` § Adding a lint rule (step 6) | `[ev: go/internal/lint/rules/licensegate/testdata/truth-separator-literal.txt]` |
| Δ2 | **The golden format cannot hold an empty directory.** A recorded case on an empty tree replays as a missing root (`not a directory`, exit 3) instead of the K20 row. Pin an empty directory in a direct test (`TestEmptyDir` here), or teach `golden` a `-- <dir>/ (dir) --` section if a second port needs it. | `go/internal/lint/rules/golden` | `[ev: go/internal/lint/rules/licensegate/licensegate_test.go]` |

## Lessons
- A recorder run through a copy of the bats suite (the lint variable pointed at the wrapper, one id per test)
  records every readable-tree run without touching the suite; chmod-000 runs belong to `golden.Unreadable`.
- zsh does not word-split an unquoted `$args`: a byte-identity loop must split explicitly, or a `--strict <root>`
  run silently measures a usage error on both sides.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-lints-a`) — Δ1 and Δ2 `BUILD-LOOP.md` § 5 kit tooling doctrine (truth case RED for its reason; golden cases from the replay).
