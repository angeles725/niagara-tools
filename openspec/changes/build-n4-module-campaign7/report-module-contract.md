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

## Expected report — ColdRoomPan jar-mode baseline (B798 @ kit v0.17.0)
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
This is the B798 jar-mode run (without `--src`). The shipped script runs `verify-module.sh --src`, which adds
typecount (PASS), facets (PASS), and stored (SKIP) rows — legitimately expanding the jar-mode baseline.

## Shipped `--src` mode output (ColdRoomPan-rt, kit v0.18.0, 2026-09-05)
```
ColdRoomPan-rt  PASS  bytecode  8 classes, all major 52
ColdRoomPan-rt  PASS  signed  META-INF/NIAGARA4.SF present
ColdRoomPan-rt  PASS  types  6 declared types resolve to classes
ColdRoomPan-rt  PASS  baja  stamped baja 4.14 <= target 4.14
ColdRoomPan-rt  SKIP  stored  no --stored
ColdRoomPan-rt  PASS  typecount  jar declares 6 types == module-include.xml
ColdRoomPan-rt  PASS  facets  no raw-number MIN/MAX facet under .../ColdRoomPan-rt/src
ColdRoomPan-rt  PASS  palette  module.palette has 3 component entries
ColdRoomPan-rt  PASS  dup-keys  0
ColdRoomPan-rt  WARN  slot-coverage  50.0% (missing DefrostMode,FanMode,StagingMode)
ColdRoomPan-rt  PASS  timer-ticket  BDefrostController.java: timer cancelled in stopped()
ColdRoomPan-rt  FAIL  timer-ticket  BEvaporatorUnit.java: schedules a Clock ticket but stopped() does not cancel it
report-module: 1 artifact · 9 PASS · 1 FAIL · 1 WARN · 1 SKIP  ->  ISSUES
```
QA pins `qa/c7-report-module` on BEvaporatorUnit FAIL and slot-coverage WARN (exit 1); dropping either flips the pin.
(DashboardPan-ux adds a `--plano FAIL` once #47 lands — B797.)

**Note on dup-keys escalation:** `slot-coverage.sh` emits per-key WARN lines but exits 0; `report-module.sh`
collapses them into a single `dup-keys N` row escalated to FAIL when N>0 (campaign-6 doctrine: lexicon dup keys = hard fail).
This is not a mapping bug — it is intentional severity promotion by the aggregator.
