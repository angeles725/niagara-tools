<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · r10-dr5-driver-exemplar

**Session**: niagara-research quality round 10, driver-authoring task DR5, branch `feat/r10-dr5-driver-exemplar`
from DR2 (merged as #502), origin/main merged after DR3 (#503)
**Delta count**: 3

## What happened
The MinimalDriver fixture got a `BTestNgStation` loopback test and a plain-Java mock device on `127.0.0.1` port 0
that speaks the driver codec. The test covers ping ok, ping down, proxy read, write, poll timeout with a re-ping,
and the network's Context-form discovery action. A scaffold (`DemoDriver`) was built signed into a 4.14.0.162
mirror, and its tests ran through `bin\test.exe` on a scratch copy of the home: 6/6 pass. With the mock answering
device-error status the suite failed 6/6. With the mock answering the wrong sequence number it failed 5/6. The
build also exposed a DR2 defect: the `-wb` point manager's include line had no `<agent>`, so the built jar dropped
the `@AgentOn`. It is fixed, and report-module goes from 1 FAIL to CLEAN.

## Evidence
- verify-module on the scaffold: `27 passed, 0 failed, 9 skipped, 0 warned -> ALL PASS`, `PASS moduletest …
  moduleTest-include.xml present` (DR2's run had `1 warned`, the missing moduleTest).
- GREEN: `Total tests run: 6, Failures: 0, Skips: 0`; RED (status 1): `Total tests run: 6, Failures: 6`; RED
  (tag + 1): `Failures: 5`, the answered-ping case passed.
- First setup failed: `java.lang.ArrayIndexOutOfBoundsException: 0 at
  javax.baja.test.BTestNgStation.startStation(BTestNgStation.java:192)` with `makeDefaultAuthFoxServices = false`.
- `RunNiagaraTestTask` (niagara-plugins-7.6.17.jar) resolves `niagaraHome.file("bin/test")` and passes
  `-groups:` / `-excludegroups:` / `-output:` / `-v:`.
- A symlink mirror over `\\wsl.localhost`: `dir …\home414\bin\nreVersion.xml` → "No se encuentra el archivo";
  `modules\baja.jar` lists as `<JUNCTION>`. With a UNC user home: `FileLockException: error obtaining lock for
  Z:\…\winuser\security\.km` → `NreInstantiationException`.
- Boot: `Rebuilt: 14206 types [289695ms]`, `Niagara runtime booted … (972491ms)`. A later boot logged
  `Out-of-date: Module added "nmodsreflow.77-rt"` and rebuilt again until that jar was removed from the copy.
- Unsigned trust: `Could not validate certificate path for entry … in module DemoDriver-rtTest: Could not validate
  cert chain`, and the tests still ran.
- Signing: a key without EKU failed `:jar` with `Certificate for NIAGARA4 is not valid for code signing`.
- The built `-wb` jar before the fix: `<type class="com.angeles.DemoDriver.wb.BDemoDriverPointManager"
  name="DemoDriverPointManager"/>` with no `<agent>`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Driver station test in the scaffold. How to run `bin/test` from WSL: copy the home, use a local user home, keep Fox on a free port, count the tests, not the exit code | `fixtures/MinimalDriver`; `types/driver-authoring.md` § 12.1; `types/moduleTest.md` § TEST-G8 | `[ev: code docSource BTestNgStation.java :188-192]` |
| Δ2 | The network's Context-form `doSubmitDiscoveryJob` is the bound do method, checked live | `types/driver-authoring.md` § 12.1 | `[ev: corpus B1239 §1239.11]` |
| Δ3 | A mock that answers with tag + 1 still completes back-to-back pings (tag aliasing) | `types/driver-authoring.md` § 12.1 | `[ev: corpus B1239 §1239.6]` |

## Owed
- G33 `run-niagara-test` (Go) can now follow a proven route: copy the home, set `niagara_user_home`, run
  `bin\test.exe <module>`, read the summary line.
- The run is local only: CI has no Windows NRE.

## Lessons
- A test that passes under a mutation is either missing its teeth or the mutation isn't what you meant to test.
  Here the answered-ping case needed a fresh `lastOkTime`, because a new device starts out not down.
- Build the scaffold the kit emits and run its lints. DR2's dropped agent showed up only in report-module.

---
**Status**: FOLDED 2026-10-24 (branch `feat/r10-dr5-driver-exemplar`). Δ1 `fixtures/MinimalDriver`,
`types/driver-authoring.md` § 12.1, `types/moduleTest.md` § TEST-G8; Δ2 and Δ3 § 12.1. The INDEX row is generated
from the header.
