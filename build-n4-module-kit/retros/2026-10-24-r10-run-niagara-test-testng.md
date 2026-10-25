<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-run-niagara-test-testng

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test` phase 2a (the verdict from
`testng-results.xml`), branch `r10/run-niagara-test-testng` from origin/main 5cc5dd07
**Delta count**: 2

## What happened
A live run of a station test with `bin\test.exe <module> -v:2 -output:<dir>` printed no `Total tests run` line:
the console held only thread dumps, so the phase-1 console verdict could only say FAIL (no summary). The
verdict of that run lives only in `<dir>/testng-results.xml`. Phase 2a adds `verdict --testng <xml> [<log>]`
(`go/internal/niagaratest` `TestNG`, `TestNGFile`).

- **Rule.** PASS only when the root is `<testng-results>` with total > 0, failed = 0 and skipped = 0, no
  `test-method` has status FAIL (config methods included), the document is well-formed to its end, and the
  root counts agree with the non-config test-method statuses (and total = passed + failed + skipped + ignored).
  A missing file, malformed or truncated XML, another root, a missing or non-count root attribute, an unknown
  status or a file over 64 MiB is FAIL with its own reason. Rows name the failed test-methods, 20 at most, then
  "and N more".
- **Config methods.** TestNG leaves `is-config="true"` methods out of the root counts, so a failed
  `setupStation` shows as `failed="0" skipped="N"`. Skips are therefore a FAIL in this mode ("N skipped (a failed
  setup skips its tests)"), and a FAIL config method is its own reason.
- **Console log.** When given, its phase-1 reasons (`invalid test target(s)`, `failed signature validation`,
  failure counts) still fail the run; its missing summary line is no reason in this mode.
- **Bounded memory.** A streaming `encoding/xml` decoder; the file size is checked against `MaxTestNGBytes`
  before the read and the read is capped again with `io.LimitReader`. Nothing is sized from the input.
- **Tests.** A table of 9 XML edges (empty, another root, total 0, malformed, truncated, counts that disagree,
  a total that is not the sum, bad root attributes, an unknown status) plus a sparse-file size-cap test; 7 new
  CLI goldens (green, a failure, a config failure plus skips, a missing file, 22 failures, a console signature
  rejection with a green XML, `--testng` with no value), no case repeated between table and goldens; one bats
  smoke run (RNT3). RED was observed on a stub `TestNG`/`TestNGFile` (9/9 table cases, the size cap) and the
  unchanged CLI (11/16 goldens).
- **Not done here.** The claim "test.exe exits 0 on failures" in `types/moduleTest.md` and
  `types/driver-authoring.md` is left as it is: another session is measuring it. Real anonymized
  `testng-results.xml` fixtures and the runner half of G33 are owed.

## Evidence
- `go test ./internal/niagaratest` green: TestNG table, size cap, 16 goldens `[ev: go/internal/niagaratest/testng_test.go]`
- bats RNT1-RNT3 green `[ev: tests/run-niagara-test.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The console of `test.exe <module> -v:2 -output:<dir>` carries no `Total tests run` summary (only thread dumps): judge the run from `<dir>/testng-results.xml` with `run-niagara-test verdict --testng`, never from the console | `types/moduleTest.md` § Known gotchas; `BUILD-LOOP.md` § 5 | `[ev: go/internal/niagaratest/testng.go]` |
| Δ2 | TestNG's root counts leave config methods out: a failed `@BeforeClass` reads as failed=0 with skips, so a results-file verdict fails on skips and on any FAIL test-method, config included | `BUILD-LOOP.md` § 5 fail-closed bullet | `[ev: go/internal/niagaratest/testdata/testng-config-skips.txt]` |

## Lessons
- A tool that judges a run must read the artifact that carries the verdict for that invocation; when a flag
  (here `-output:`) moves the summary off the console, a console-only verdict is a permanent FAIL, not a check.
