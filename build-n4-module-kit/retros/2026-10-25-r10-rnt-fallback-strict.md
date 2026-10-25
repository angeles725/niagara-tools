<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-fallback-strict

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, the ignored-count fallback made
strict, branch `r10/rnt-fallback-strict` from 63b1c44f (origin/main)
**Delta count**: 1

## What happened
Retro r10-rnt-ignored-names named the ignored methods from the source, but `--allow-ignored-count` still skipped
the scan: picking the count flag, swapping one disabled test for another and keeping the count passed again. `run`
now scans the source under every allowance, each rule RED first on origin/main (no station boot):
1. The scan always runs, and every report prints its names (`source: the registered types disable N
   test-methods: <Class.method>, ...`).
2. Under `--allow-ignored-count` a scan refusal (class-level disable, non-literal `enabled`, unreadable or
   ambiguous source) is a NOTE row, not exit 3, and the count is judged as case (i), printed on the PASS row.
3. A source count that differs from the root `ignored` (a group exclusion, a base outside `srcTest/`) is case
   (ii), printed on the PASS row; the count must still match the root `ignored` exactly.
4. When the scan names every ignored method, the count FAILs "use --allow-ignored names" and lists them; the
   reason is a printed note only and never overrides this.
5. `verdict --testng` has no source and keeps the count pair unchanged.

## Evidence
- RED on origin/main 63b1c44f: `TestRunIgnoredNamedFromSource` — `(a) the swap under the count fallback` (exit 0,
  want 1), `(b) the count fallback with the exact count` (exit 0, want 1), both `(c) case (i)` cases and
  `(d) case (ii)` (no case row), `(f) no allowance and no disable` and the two name cases (no scanned-names row).
  GREEN after `scanIgnored` in `run.go` and `judgeCount`/`fallbackCase` in `ignored.go`
  `[ev: code go/internal/niagaratest/ignored.go]`.
- Mutation: restoring "the count skips the scan" in `scanIgnored` turned `(a)` red again (exit 0, want 1).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test run` scans the source under `--allow-ignored-count` too and prints the scanned names; the count is accepted only when the scan refused (case (i), a NOTE row instead of exit 3) or the source count differs from the root `ignored` (case (ii)), the row naming the case; a scan that names every ignored method FAILs the count "use --allow-ignored names", whatever the reason; `verdict --testng` keeps the count pair | `types/moduleTest.md` § Known gotchas; `toolbelt/run-niagara-test.sh` header | `[ev: code go/internal/niagaratest/ignored.go]` |

## Lessons
- A fallback that skips the strict check reopens the hole the check closed: run the check under the fallback
  too, and accept the fallback only for the cases the check cannot decide.

## Fold
- The gate is code in the branch that wrote this retro (`go/internal/niagaratest/ignored.go`, `run.go`,
  `run_ignoredsrc_test.go`, `ignored_test.go`).
- Δ1 is cited in `types/moduleTest.md` § Known gotchas (the count-fallback bullet). Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-fallback-strict`; log in `ledger.d/2026-10-25-r10-rnt-fallback-strict.md`) — Δ1 in `types/moduleTest.md` § Known gotchas.
