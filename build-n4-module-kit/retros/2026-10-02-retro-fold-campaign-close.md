<!-- review-status: pending -->
# 2026-10-02 · kit · retro-fold-campaign-close

**Session**: close of the 2026-10-02 retro-fold campaign (WU0-WU11, PRs #183-#206, release v0.29.0)
**Delta count**: 4

## What happened
The campaign folded the 23 retros of 2026-09-24..2026-10-01 (147 Δ) into the kit core as one PR per work unit,
each branched from the current `main`, reviewed by the native RDD review when due, and merged after CI. Four process
frictions repeated across the work units and were solved by hand each time: the kit ledger conflicted on every
merge of a parallel work unit; a parallel `bats -j` run reported success with zero tests; a feature-doc-only push was
blocked by the pre-push retro gate; and non-blocking review advisories arrived after approval, when applying them
would have reopened the review. The release brief also named a stale version (the root `VERSION` had moved on with
the MCP-kit releases).

## Evidence
- WU10 merge of `origin/main`: `BUILD-STATE.md`, `METHODOLOGY.md` and the feature doc conflicted; the ledger was
  resolved by keeping main's entries and re-applying WU10's per-entry "LANDED WU10" edits by a 3-way word merge
  `[ev: abe8d24]` `[ev: odd/tasks/fold-2026-10-02-pending-retros.md WU10 merge note]`
- `bats -j` without GNU `parallel` installed runs 0 tests and exits without a failing test line; every work unit ran
  `bats tests/*.bats` serially instead (~860 tests) `[ev: odd/tasks/fold-2026-10-02-pending-retros.md WU9 "(serial)"]`
- `.githooks/pre-push` counts every kit file except `BUILD-STATE.md` and `retros/**` as build-relevant, so a push that
  only records evidence in `odd/tasks/*.md` needs `Retro: none (trivial: …)` `[ev: .githooks/pre-push:26-34]` `[ev: 02c72a1]`
- Approved reviews returned informational advisories (WU0: 9, WU10: 4, WU11a: 7); each was routed to a follow-up
  (issue #199, the next work unit) because editing the approved candidate would invalidate its receipt
  `[ev: odd/tasks/fold-2026-10-02-pending-retros.md WU0/WU10/WU11a review notes]`
- Release brief said `0.22.0 → 0.23.0`; root `VERSION` was `0.28.1` and tags ran to `v0.28.1` `[ev: VERSION]` `[ev: git tag]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Ledger merge rule: on a `BUILD-STATE.md` conflict between parallel work units, keep the target branch's entries and re-apply the branch's edits per entry (3-way word merge at entry granularity), never take one side wholesale; then run `sweep-build-state.sh` and check for duplicate entries. | `METHODOLOGY.md` § fragment-merge rule (next to the APPEND rule) | `[ev: abe8d24]` |
| Δ2 | Test-run rule: run `bats tests/*.bats` serially unless GNU `parallel` is installed; a parallel run must assert the ok count equals `bats --count`, because `bats -j` without `parallel` runs nothing. | `CONTRIBUTING.md` § tests + `ORCHESTRATION.md` writer prompt clauses | `[ev: odd/tasks/fold-2026-10-02-pending-retros.md WU9]` |
| Δ3 | Close-gate note: a push that only records evidence in `odd/tasks/*.md` carries `Retro: none (trivial: feature-doc evidence only)`; or exempt `odd/tasks/**` from the pre-push build-relevant set. | `BUILD-LOOP.md` § 7 (trivial exit) or `.githooks/pre-push` | `[ev: .githooks/pre-push:26-34]` |
| Δ4 | Review-advisory routing: informational findings of an APPROVED review go to the feature doc and a follow-up issue or the next work unit, never into the approved candidate (an edit after approval reopens the review). | `ORCHESTRATION.md` § 7 candidate shaping | `[ev: issue #199]` |

## Lessons
- A ledger shared by parallel writers needs a merge rule at entry granularity, or every merge rewrites history by hand.
- A test runner that can run zero tests and exit 0 is not a gate; count what ran.
- A gate that cannot tell evidence from behavior asks for a waiver every time; say which waiver in the rule.
- An approved receipt is immutable; advisories are new work, not amendments.
- Read `VERSION` and `git tag` before planning a release bump; a brief's version can be stale.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-02-retro-fold-campaign-close.md | kit | 2026-10-02 | pending | 4 |`
