# report-module.sh — output contract (campaign 7, PR8)
`report-module.sh <module-root> [--target-version 4.14]`
One aggregated read-only conformance report over EVERY profile artifact (`<MOD>-{rt,ux,wb,…}`) found under
`<module-root>`. Composes the campaign-6 toolbelt; invents no new checks. Grounded in the B798 baseline run.

## Per artifact, invoke (skip a tool when its input is absent)
- `verify-module.sh --src <artifact> --target-version <v> <build/libs/*.jar>`  — SKIP if no built jar.
- `slot-coverage.sh` parse (`module-include.xml` + `module.lexicon`).
- dup-lexicon-keys over `module.lexicon`.
- `lint-timers.sh <artifact>/src`.
- `--plano` (the B797 four-value check) ONLY when an `index.html` exists under `<artifact>/src/rc/`.

## Row format — reuse each tool's own row, prefixed by the artifact
```
<artifact>  PASS|FAIL|WARN|SKIP  <check>  <detail>
```
Severity mapping: verify-module FAIL→FAIL; lint-timers FAIL→FAIL; dup-keys >0→FAIL; `--plano` mismatch→FAIL;
slot-coverage <100%→WARN; empty palette→WARN (SKIP if the artifact is scaffold-only / has no jar).

## Summary + exit
```
report-module: <N> artifacts · <p> PASS · <f> FAIL · <w> WARN · <s> SKIP  ->  <CLEAN|ISSUES>
```
Exit: **0** clean (zero FAIL) · **1** any FAIL · **3** env (no JDK 8 / not a niagara_home).

## Expected report — ColdRoomPan TODAY (real-tree evidence, B798 @ kit v0.17.0, the QA pin)
```
ColdRoomPan-rt  PASS  bytecode      8 classes, all major 52
ColdRoomPan-rt  PASS  signed        META-INF/NIAGARA4.SF present
ColdRoomPan-rt  PASS  types         6 declared types resolve
ColdRoomPan-rt  PASS  baja          stamped baja 4.14 <= 4.14
ColdRoomPan-rt  PASS  palette       3 component entries
ColdRoomPan-rt  PASS  dup-keys      0
ColdRoomPan-rt  PASS  timer-ticket  BDefrostController.java: timer cancelled in stopped()
ColdRoomPan-rt  FAIL  timer-ticket  BEvaporatorUnit.java: schedules a Clock ticket but stopped() does not cancel it
ColdRoomPan-rt  WARN  slot-coverage 50.0% (missing DefrostMode,FanMode,StagingMode)
report-module: 1 artifact · 7 PASS · 1 FAIL · 1 WARN · 0 SKIP  ->  ISSUES   (exit 1)
```
QA pins `qa/c7-report-module` on this exact ColdRoomPan-rt output; dropping the BEvaporatorUnit FAIL or the
slot-coverage WARN flips it. (DashboardPan-ux adds a `--plano FAIL` once #47 lands — B797.)
