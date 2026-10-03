<!-- review-status: pending -->
# 2026-10-03 · kit · deferred-lints-2026-10-03-close

**Session**: close of the deferred-lints feature `odd/tasks/deferred-lints-2026-10-03.md` (D1-D5b, PRs #230-#244, release v0.31.0)
**Delta count**: 5

## What happened
The feature folded the polish close retro, closed the #226 lead fail-open (a `find` error skipped files in six lints)
through a shared `lib/scan-files.sh`, and built the three deferred lint candidates (`lint-size`,
`lint-license-isoperational-gate`, `lint-set-null-ord`) plus the project-level `split-package-check` of #142. Every PR
ran a granted 4-lens native review and was approved. Under the anti-cascade policy each review still surfaced exactly
one fail-open in the code it had just added, which became the next sub-task (D1b, D1c, D2b, D3b, D4b, D5b). The chain
converged: the D5b review returned suggestions only. Three mechanical traps cost time: the first capture round of D1
returned `invalid_request` (no `--agent` in the capture tokens), four observed mutations were the wrong shape and
"passed" for the wrong reason, and two overlapping fail-closed guards masked each other's mutation.

## Evidence
- D1 captures: `invalid_request` "requires … either --input or --agent"; the bound STATUS had been reached from the
  `next_transition.command` of `gentle-ai review assess`, which carries no `--agent`; the canonical preflight with
  `--agent claude-code` returned the same lineage's START, whose replay bound the agent `[ev: odd/tasks/deferred-lints-2026-10-03.md D1]`
- Wrong-shape mutations: `if ! cmd || true; then` (parsed as `(! cmd) || true`, always true) for LSZ-finderr,
  SPC-findtree and SPC-aggfail; an awk mutation that broke the script instead of the guard for LSZ-awkfail; a
  quote-state mutation that fell through to the unquoted branch for SH1-squote `[ev: odd/tasks/deferred-lints-2026-10-03.md D1b D2 D5 D5b]`
- Masked guard: SPC-finderr did not bite while the module search still descended into `src/` (it raised exit 3 first);
  it bit once the search pruned `src/` `[ev: odd/tasks/deferred-lints-2026-10-03.md D5]`
- One fail-open per review, each folded into the next task's PR when it touched the same file or helper: D1b, D1c, D2b,
  D3b, D4b, D5b; D5b (own PR) closed the chain `[ev: issue #226]`
- Real-tree smoke runs (a client checkout, decompiled vendor modules, read only) gave the noise level of each new
  advisory lint before release: lint-size 10 genuine rows; license gate rows on two licensed vendor classes;
  set-null-ord 81 rows over 334 vendor files; split-package 0 rows on one checkout, every package on a directory of 16
  checkouts `[ev: odd/tasks/deferred-lints-2026-10-03.md D2-D5]`
- Observed, not fixed: 12 other lints still walk with `find … 2>/dev/null` (same class as the #226 lead item) `[ev: issue #226]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | RDD preflight: always start from the canonical `gentle-ai review status --cwd <repo> --contract … --agent <runtime> --next-transition` (with `--base-ref`/`--committed-only` when assessing a commit), not from the `next_transition.command` that `review assess` returns — that command carries no `--agent`, so the capture tokens come back without it and every capture fails `invalid_request`. If it already happened: re-run the canonical preflight; it returns the same lineage's START, and its exact replay binds the agent. | `ORCHESTRATION.md` § 4 Escalation gate | `[ev: odd/tasks/deferred-lints-2026-10-03.md D1]` |
| Δ2 | Observed-mutation shape: neutralize a status check as `if ! { cmd || true; }; then` or by deleting the `had_err=1` / `exit 3` line — never `if ! cmd || true; then` (always true). A mutation counts only when the mutated script still parses and the failing assertion is the targeted one; run the mutated script once by hand when the RED looks too easy. | `CONTRIBUTING.md` § 2 TDD requirement | `[ev: odd/tasks/deferred-lints-2026-10-03.md D2 D5]` |
| Δ3 | When a new guard's mutation does not bite, look for an earlier fail-closed guard that catches the same fixture first, and separate their inputs (e.g. prune what the earlier pass does not need) so each guard is pinned by its own test. | `METHODOLOGY.md` § Conformance rules | `[ev: odd/tasks/deferred-lints-2026-10-03.md D5]` |
| Δ4 | Anti-cascade in practice: the fail-open sub-task a review opens rides in the next task's PR when it touches the same file or helper (one review per PR), and in its own PR at the end of the chain; record which. | `ORCHESTRATION.md` § 7 | `[ev: issue #226]` |
| Δ5 | Before releasing a new advisory lint, smoke it read-only on one real client checkout and on decompiled vendor code, and record the row count and a sample verdict in the feature doc — it is the cheapest evidence of its noise level (here: every lint-size row genuine, Tridium's own importers matching set-null-ord, and the one-checkout scope rule of split-package-check). | `BUILD-LOOP.md` § 5 pre-gate | `[ev: odd/tasks/deferred-lints-2026-10-03.md D2-D5]` |

## Lessons
- A review that approves still finds the fail-open in the newest code; plan one fix-forward slot per lint.
- A mutation that "bites" because the script broke proves nothing.
- Two fail-closed guards on one fixture pin only the first.
- The assess command is a preflight hint, not the canonical preflight.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-03-deferred-lints-2026-10-03-close.md | kit | 2026-10-03 | pending | 5 |`
