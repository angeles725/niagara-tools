<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · o12-logic-series

**Session**: opportunity O12 (offline logic check): let a module author check pure rt control logic against a
scripted input series with expected outputs before the lab.
**Delta count**: 2

## What happened
Pure cores (client A's compressor control is the reference) were tested one scenario per JUnit method, each spelling
its inputs and timing in Java. A scenario over time reads better as a table checked against the spec line by line.
The kit had no place for Java helpers a module test could use without copying: `run-pure-test.sh` compiled the
test with `-sourcepath <rt>/src:<test root>` only. The smallest place is a kit Java source root
(`toolbelt/java`) that the launcher names in `RUN_PURE_TEST_KIT_JAVA` and the Go runner appends to the source path
(javac compiles only the classes a test references, so other modules are unaffected). The helper is Java 8 source
with no dependency; a mismatch is an `AssertionError` (JUnit's failure, exit 1) and a malformed series an
`IllegalArgumentException` before any step.

## Evidence
- RED before GREEN: every `tests/logic-series.bats` case failed on origin/main (package `n4kit.series` not on the
  source path, exit 3); all 13 pass with the helper and the source-path wiring; `tests/run-pure-test.bats` P1-P7
  and S24 stay green. `[ev: tests/logic-series.bats]` `[ev: tests/run-pure-test.bats]`
- `go/internal/puretest` TestKitJavaOnSourcePath pins the appended root; TestPass/TestSourceRoot pin the unchanged
  source path when the variable is unset. `[ev: go/internal/puretest/run_test.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. Logic series recipe: CSV format, adapter, results, and how it complements LP1 | `build-verify.md` § Logic series; `types/logic.md` § Pure-class extraction | `[ev: tests/logic-series.bats]` |
| Δ2 | APPLIED 2026-11-16. Kit Java helper root on the run-pure-test source path (`RUN_PURE_TEST_KIT_JAVA`) | `toolbelt/run-pure-test.sh`, `go/internal/puretest/run.go` | `[ev: go/internal/puretest/run_test.go]` |
