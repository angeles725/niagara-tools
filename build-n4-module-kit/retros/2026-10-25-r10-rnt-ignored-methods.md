<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-ignored-methods

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test` (`verdict --testng` and `run`), a gate
on ignored TestNG methods, branch `r10/rnt-ignored` from f9eabe6e (origin/main)
**Delta count**: 1

## What happened
A real station run of a client root printed `testng: total 43, passed 21, failed 2, skipped 0`. The 20 missing
tests were `@Test(enabled = false)` methods: TestNG counts them in the root `ignored` and `total` attributes only,
and the runner checked `total = passed + failed + skipped + ignored` but never failed on `ignored`. Once the 2
failures were fixed, the run would have read green with 20 tests silently off. The gate, each rule RED first on
the merged origin/main runner (no station boot):
1. A root `ignored` above 0 is a FAIL reason naming the count, and the count row gains `, ignored N`.
2. `--allow-ignored-count <n> --allow-ignored-reason <why>` (both or neither, on `verdict --testng` and `run`)
   waives exactly n; a PASS row prints the reason. More ignored than n (a new disable) and fewer (a stale
   allowance) are each FAIL: a blanket allowance never hides a new disable.
3. A count with no or a blank reason, a reason with no count, a count of 0 or no number, a reason that is not one
   printable line of at most 200 bytes, a flag given twice and an allowance without `--testng` are exit 2 (usage),
   before any file is read or anything copied.
4. The PASS verdict row counts `total - ignored` as tests run.

## Evidence
- The format: the TestNG bundled in a 4.15 install's `modules/test-wb.jar` (`javap -c` in a scratch dir, removed).
  `XMLReporter` adds `ITestContext.getExcludedMethods().size()` to the root `ignored`; `XMLSuiteResultWriter`
  writes `test-method` elements only from `getPassedTests`, `getFailedTests`, `getSkippedTests` and
  `getFailedButWithinSuccessPercentageTests`. So ignored methods are never named, and the allowance is a count
  with a reason, not a list of `Class.method` names.
- RED, then GREEN: `TestVerdictIgnored` (`ignored without an allowance`: exit 0, want 1), `TestVerdictAllowanceRefusals`,
  `TestRunIgnored` (`ignored without an allowance`: exit 0, want 1; `allowed`: exit 2, want 0), bats RNT9
  `[ev: code go/internal/niagaratest/ignored.go]`.
- Mutation: removing the `judgeIgnored` call from `counts` turned `TestVerdictIgnored`, `TestRunIgnored` and
  `TestTestNG/total is not the sum` red.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test verdict --testng` and `run` FAIL a results file whose root `ignored` is above 0 unless `--allow-ignored-count <n> --allow-ignored-reason <why>` waives exactly n (the reason printed); more or fewer ignored than allowed is FAIL, a count with no reason exit 2. testng-results.xml names no ignored method, so the allowance is a count, not names | `types/moduleTest.md` § Known gotchas; `toolbelt/run-niagara-test.sh` header | `[ev: code go/internal/niagaratest/ignored.go]` |

## Lessons
- A count that a sum check accepts is not a count that was judged: `total = passed + failed + skipped + ignored`
  held for 43 = 21 + 2 + 0 + 20, and the 20 never ran.

## Fold
- The gate is code in the branch that wrote this retro (`go/internal/niagaratest/ignored.go`, `ignored_test.go`).
- Δ1 is cited in `types/moduleTest.md` § Known gotchas (the ignored-methods bullet). Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-ignored`; log in `ledger.d/2026-10-25-r10-rnt-ignored-methods.md`) — Δ1 in `types/moduleTest.md` § Known gotchas.
