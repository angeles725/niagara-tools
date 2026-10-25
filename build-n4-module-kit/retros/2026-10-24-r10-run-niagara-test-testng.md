<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-24 · kit · r10-run-niagara-test-testng

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test` phase 2a (the verdict from
`testng-results.xml`), branch `r10/run-niagara-test-testng` from origin/main 5cc5dd07
**Delta count**: 4

## What happened
A live station-test run of `bin\test.exe <module> -v:2 -output:<dir>` first looked summary-less: the captured
stdout held only shutdown thread dumps. The finished run showed where the summary goes: test.exe prints its
`Total tests run: N, Failures: F, Skips: S` lines (one per suite, then `Results of all run tests`) on stderr.
`<dir>/testng-results.xml` is the authoritative record. Phase 2a adds `verdict --testng <xml> [<log>]`
(`go/internal/niagaratest` `TestNG`, `TestNGFile`); the console log, when given, is the cross-check.

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
  failure counts) still fail the run; its missing summary line is no reason in this mode (a stdout-only capture
  has none).
- **Exit code, measured.** test.exe exits 0 when every test passes and -1 (255 or 4294967295, by reader) when
  any fails. The earlier "exits 0 on failures" readings came from `cmd /c "… & echo %ERRORLEVEL%"`, which
  expands `%ERRORLEVEL%` when the line is parsed, before test.exe runs; `cmd /v:on` with `!ERRORLEVEL!` reads
  the real code. The output verdict stays as belt and braces.
- **Fox on loopback.** The Windows NRE never binds Fox to 127.0.0.1 alone, so a loopback-only station test turns
  Fox off (`foxEnabled`, `foxsEnabled`, multicast, announcement) and its fixture tolerates only
  `BTestNgStation`'s exact `Fox service did not start in time` failure, only while the station is running.
- **Bounded memory.** A streaming `encoding/xml` decoder; the file size is checked against `MaxTestNGBytes`
  before the read and the read is capped again with `io.LimitReader`. Nothing is sized from the input.
- **Tests.** A table of 9 XML edges (empty, another root, total 0, malformed, truncated, counts that disagree,
  a total that is not the sum, bad root attributes, an unknown status) plus a sparse-file size-cap test; 9 new
  CLI goldens: 7 synthetic (green, a failure, a config failure plus skips, a missing file, 22 failures, a console
  signature rejection with a green XML, `--testng` with no value) and 2 from the real run (green, 6 failed),
  anonymized, with no case repeated between table and goldens; one bats smoke run (RNT3). RED was observed on a
  stub `TestNG`/`TestNGFile` (9/9 table cases, the size cap) and the unchanged CLI (11/16 goldens).
- **Phase-1 count fix.** The console PASS row summed every summary line, so the real stderr (3 + 3 + 5 per
  suite, then the 11 of `Results of all run tests`) read "22 tests run". A summary right after that header is
  now marked `GrandTotal` (its row says "(all run tests)") and the PASS row reports it; without the block the
  per-suite lines are still summed, and every summary line still fails the run on its own. RED: the
  `pass-grand-total` golden (the real stderr) printed "22 tests run", and two table cases lacked the mark.
- **Not done here.** The runner half of G33 is owed.

## Evidence
- `go test ./internal/niagaratest` green: TestNG table, size cap, 19 goldens `[ev: go/internal/niagaratest/testng_test.go]`
- the real run, anonymized: `[ev: go/internal/niagaratest/testdata/real-red.txt]`
- bats RNT1-RNT3 green `[ev: tests/run-niagara-test.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | test.exe prints its `Total tests run` summaries on stderr; stdout holds only thread dumps. Capture stderr (`2>&1`), judge `<dir>/testng-results.xml` (`-output:`) as the authoritative verdict with `run-niagara-test verdict --testng`, the console summary as a cross-check | `types/moduleTest.md` § Known gotchas; `BUILD-LOOP.md` § 5 | `[ev: go/internal/niagaratest/testdata/real-green.txt]` |
| Δ2 | TestNG's root counts leave config methods out: a failed `@BeforeClass` reads as failed=0 with skips, so a results-file verdict fails on skips and on any FAIL test-method, config included | `BUILD-LOOP.md` § 5 fail-closed bullet | `[ev: go/internal/niagaratest/testdata/testng-config-skips.txt]` |
| Δ3 | test.exe exits -1 when a test fails and 0 when all pass; `%ERRORLEVEL%` in a `cmd /c` one-liner expands before the run and reads 0, so read `!ERRORLEVEL!` under `cmd /v:on` | `types/moduleTest.md` § Known gotchas; `types/driver-authoring.md` § 12.1 | `[ev: types/driver-authoring.md § 12.1]` |
| Δ4 | The Windows NRE never binds Fox to loopback only: a loopback-only station test turns Fox off (foxEnabled, foxsEnabled, multicast, announcement) and tolerates only BTestNgStation's exact "Fox service did not start in time" failure while the station runs; otherwise the free-port step stands | `types/driver-authoring.md` § 12.1 | `[ev: types/driver-authoring.md § 12.1]` |

## Lessons
- Before calling an output summary-less, capture every stream the tool writes: the summary was on stderr all
  along, and a stdout-only capture made a working run look broken.
- A `cmd /c` one-liner that echoes `%ERRORLEVEL%` reports the code from before the command ran; measure exit
  codes with delayed expansion before writing them down as a gotcha.

## Fold
- The verdict is code already shipped in the PR that wrote this retro (`go/internal/niagaratest/testng.go`, `verdict --testng`); not re-implemented.
- Δ1 and Δ3 were cited at merge in `types/moduleTest.md` § Known gotchas (Δ3 also in `types/driver-authoring.md` § 12.1); Δ4 in `types/driver-authoring.md` § 12.1.
- Δ2 → `BUILD-LOOP.md` § 5 fail-closed bullet (`testdata/testng-config-skips.txt`); the moduleTest gotcha already states the case for test.exe.
- "Not done here" (the runner half of G33) shipped as retro r10-run-niagara-test-run. Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `docs/r10-fold-tail-4`; log in `ledger.d/2026-10-25-r10-fold-tail4.md`) — Δ1, Δ3 in `types/moduleTest.md` § Known gotchas; Δ2 in `BUILD-LOOP.md` § 5 fail-closed bullet; Δ4 in `types/driver-authoring.md` § 12.1.
