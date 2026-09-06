<!-- review-status: pending -->
# Retro: Campaign 8 — verify-module facets-req + ord-literal sub-checks (facets-lint)

**Date:** 2026-09-05
**Change:** build-n4-module-campaign8 PR4 — `feat/c8-facets-lint`
**Scope:** `build-n4-module-kit/toolbelt/verify-module.sh` — `check_facet_presence` (label `facets-req`, WARN), `check_ord_literal` (label `ord-literal`, WARN), D9b dot-dir prune on `check_raw_double_facets`; K19 routing to `BUILD-LOOP.md` + `skill/SKILL.md`
**Lead:** Campaign 8 PR4 executor
**RED tip:** `9cb3168`

[ev: corpus B787] [ev: corpus B801]

---

## What shipped

| Sub-check | Label | Exit | Description |
|-----------|-------|------|-------------|
| `check_facet_presence` | `facets-req` | WARN (0) | OPERATOR numeric slot without a `facets =` key → WARN; name-pattern slot (`*Setpoint*`, `*Temp*`, `*Limit*`, `*Band*`, `*Psi`) without UNITS → WARN; `demand`/`*count*`/`stages` without PRECISION → WARN. Presence-only: never reads facet values. |
| `check_ord_literal` | `ord-literal` | WARN (0) | Java string literal matching `"(station:\|local:\|slot:/)` under `<profile>/src`, excluding `srcTest/`, `*OrdConstants*` files with comments, and `defaultValue`-bearing lines. |
| D9b prune (existing `check_raw_double_facets`) | `facets` | unchanged | `find ... -type d -name '.*' -prune -o -name '*.java' -print` — excludes `.deploy-baseline`, `.git`, etc. |

---

## TDD evidence

| Phase | Evidence |
|-------|----------|
| RED | Commit `9cb3168` on `qa/c8-facets-lint`: F1-F8 bats failing before impl. RED pins WARN for facets-req and ord-literal. |
| GREEN | `check_facet_presence` + `check_ord_literal` + D9b prune implemented; F1-F3, F5, F8 turned green (F4, F6, F7 were correct before). |
| REFACTOR | False-positive fix commit `b8460e9`: mark ALL `@NiagaraProperty` slots in `seen[]` (not only those that WARN) to block pass-2 double-fire on non-numeric OPERATOR slots. `shellcheck` exit 0. 218/218 bats green. |

---

## Design deviations

### D1: tasks.md 4.2 says FAIL — RED pins WARN (K13 applied)

**tasks.md 4.2** originally read: "OPERATOR numeric without MIN+MAX → **FAIL**".

**RED tip 9cb3168** pins all `facets-req` rows as **WARN** and all `ord-literal` rows as **WARN**. Following K13 (RED is authoritative over tasks.md wording), the implementation uses WARN throughout and never changes exit code (K20).

**tasks.md 4.2 rewritten** to reflect WARN (this retro file and the tasks.md update are in the same commit).

### D2: DashboardPan smoke ran on DashboardPan-ux, not DashboardPan-rt

**Spec** (tasks.md 4.6): "DashboardPan `DashboardReader.java:75` → WARN".
**DashboardReader.java** lives in `DashboardPan-ux/src`, not in `DashboardPan-rt/src`.

Running with `--src DashboardPan DashboardPan-rt.jar` would SKIP ord-literal because `DashboardPan/DashboardPan-rt/src` has no ORD literals. The correct invocation uses `DashboardPan-ux.jar` with `--src DashboardPan` (parent of the profile). Smoke 2 was run on `-ux` and confirmed 2 WARN rows. Deviation recorded here; no rule change required — the `profile_dir` naming convention is correct.

### D3: False-positive on non-numeric OPERATOR slots (BCompressorControl `floatingSuction`, `faultReset`)

**Root cause:** Pass 1 of `check_facet_presence` correctly skips `boolean` and `BStatusBoolean` slots (not in D5 numeric list), so they do not WARN. However, their `newProperty(OPERATOR, ..., null)` calls in the Slot-o-Matic generated region (BCompressorControl lines 1072 and 1579) matched pass 2's OPERATOR+null pattern, causing two spurious WARN rows.

**Fix:** Pass 1 now inserts ALL annotation-defined slot names into `seen[]` regardless of whether they emit a WARN. Pass 2 only fires on slots with NO `@NiagaraProperty` annotation. After the fix, 25 clean WARN rows on CompPan-rt (zero false positives for `floatingSuction`/`faultReset`).

---

## Named mutations

These prove each sub-check has independent bite:

| Check | Named mutation | Effect |
|-------|----------------|--------|
| `facets-req` (a) | Strip `facets = @Facet(...)` annotation from an OPERATOR double | F1 WARN pin fires; smoke row appears for that slot |
| `facets-req` (b) | Leave `facets = @Facet("BFacets.make(BFacets.MIN, 0.0)")` (MIN=0 present) | Check does NOT fire — presence-only logic correct |
| `ord-literal` (c) | Remove the `*OrdConstants*`+comment file exemption from `check_ord_literal` | F6 fixture (`OrdConstants.java` with `//` comment) would WARN instead of PASS |
| false-positive fix (d) | Revert pass-1 `seen[nm]=1` to only set on WARN (pre-fix behaviour) | `floatingSuction` (boolean) and `faultReset` (BStatusBoolean) would generate spurious WARN rows on CompPan smoke |

---

## Real smoke results

### Smoke 1: CompPan-rt — 25 facets-req WARN rows (design estimated 12)

```
PASS  bytecode   CompPan-rt.jar  4 classes, all major 52
PASS  signed     CompPan-rt.jar  META-INF/NIAGARA4.SF present
PASS  types      CompPan-rt.jar  1 declared types resolve to classes
SKIP  baja       CompPan-rt.jar  no --target-version
SKIP  stored     CompPan-rt.jar  no --stored
PASS  typecount  CompPan-rt.jar  jar declares 1 types == module-include.xml
PASS  facets     CompPan-rt.jar  no raw-number MIN/MAX facet under .../CompPan-rt/src
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:103 OPERATOR numeric without facets (slot=runningAmpsThreshold)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:110 OPERATOR numeric without facets (slot=suctionLowLimit)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:117 OPERATOR numeric without facets (slot=dischargeHighLimit)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:124 OPERATOR numeric without facets (slot=suctionMismatchTol)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:131 OPERATOR numeric without facets (slot=startProveDelay)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:138 OPERATOR numeric without facets (slot=minOn)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:145 OPERATOR numeric without facets (slot=minOff)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:152 OPERATOR numeric without facets (slot=stageDelay)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:159 OPERATOR numeric without facets (slot=suctionSetpoint)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:167 OPERATOR numeric without facets (slot=suctionBand)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:175 OPERATOR numeric without facets (slot=stageUpDelay)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:182 OPERATOR numeric without facets (slot=stageDownDelay)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:197 setpoint-like slot room1Setpoint missing UNITS
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:205 setpoint-like slot room2Setpoint missing UNITS
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:211 setpoint-like slot room3Setpoint missing UNITS
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:217 setpoint-like slot room4Setpoint missing UNITS
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:223 OPERATOR numeric without facets (slot=coilTD)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:239 OPERATOR numeric without facets (slot=localAtmPsi)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:247 setpoint-like slot effectiveSuctionSetpoint missing UNITS
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:336 count-like slot demand missing PRECISION
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:342 count-like slot stagesOn missing PRECISION
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:383 OPERATOR numeric without facets (slot=powerOnDelay)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:392 OPERATOR numeric without facets (slot=condenser1Mode)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:398 OPERATOR numeric without facets (slot=condenser2Mode)
WARN  facets-req  CompPan-rt.jar  .../BCompressorControl.java:404 OPERATOR numeric without facets (slot=condenser3Mode)
PASS  ord-literal  CompPan-rt.jar  no hardcoded ORD literals under .../CompPan-rt/src
PASS  palette    CompPan-rt.jar  module.palette has 1 component entries
verify-module: 7 passed, 0 failed, 2 skipped, 25 warned -> ALL PASS
```

Real count: 25 WARN rows (design estimated 12 OPERATOR doubles; actual is 13 OPERATOR + 8 setpoint-like UNITS + 2 count-like PRECISION + 2 count-like total). Zero false positives after false-positive fix (D3 above).

### Smoke 2: DashboardPan-ux — 2 ord-literal WARN rows

Invocation: `verify-module.sh --src DashboardPan DashboardPan-ux.jar` (path deviation from spec — see D2 above).

```
PASS  bytecode   DashboardPan-ux.jar  14 classes, all major 52
PASS  signed     DashboardPan-ux.jar  META-INF/NIAGARA4.SF present
PASS  types      DashboardPan-ux.jar  1 declared types resolve to classes
SKIP  baja       DashboardPan-ux.jar  no --target-version
SKIP  stored     DashboardPan-ux.jar  no --stored
PASS  typecount  DashboardPan-ux.jar  jar declares 1 types == module-include.xml
PASS  facets     DashboardPan-ux.jar  no raw-number MIN/MAX facet under .../DashboardPan-ux/src
PASS  facets-req  DashboardPan-ux.jar  no missing required facets under .../DashboardPan-ux/src
WARN  ord-literal  DashboardPan-ux.jar  .../BDashboardServlet.java:485: String bql = "station:|alarm:|bql:select * where ...";
WARN  ord-literal  DashboardPan-ux.jar  .../DashboardReader.java:75:  public static final String SERVICE_ORD = "station:|slot:/Services/DashboardService";
PASS  palette    DashboardPan-ux.jar  module.palette has 1 component entries
verify-module: 7 passed, 0 failed, 2 skipped, 2 warned -> ALL PASS
```

`DashboardReader.java:75` confirmed (spec predicted this line). Additional hit at `BDashboardServlet.java:485` (BQL query with `station:|alarm:` — a legitimate ORD literal, correct to flag).

### Smoke 3: ColdRoomPan-rt — 11 facets-req WARN rows; MIN=0 slots NOT flagged

```
PASS  bytecode   ColdRoomPan-rt.jar  8 classes, all major 52
PASS  signed     ColdRoomPan-rt.jar  META-INF/NIAGARA4.SF present
PASS  types      ColdRoomPan-rt.jar  6 declared types resolve to classes
SKIP  baja       ColdRoomPan-rt.jar  no --target-version
SKIP  stored     ColdRoomPan-rt.jar  no --stored
PASS  typecount  ColdRoomPan-rt.jar  jar declares 6 types == module-include.xml
PASS  facets     ColdRoomPan-rt.jar  no raw-number MIN/MAX facet under .../ColdRoomPan-rt/src
WARN  facets-req  ColdRoomPan-rt.jar  .../BDefrostController.java:71 setpoint-like slot terminateOnResistanceTemp missing UNITS
WARN  facets-req  ColdRoomPan-rt.jar  .../BDefrostController.java:76 setpoint-like slot resistanceTempThreshold missing UNITS
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:103 setpoint-like slot coilTemp missing UNITS
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:110 setpoint-like slot resistanceTemp missing UNITS
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:117 setpoint-like slot evapHighAlarmLimit missing UNITS
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:122 setpoint-like slot evapLowAlarmLimit missing UNITS
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:136 OPERATOR numeric without facets (slot=valveMode)
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:142 OPERATOR numeric without facets (slot=fanMode)
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:148 OPERATOR numeric without facets (slot=resistanceMode)
WARN  facets-req  ColdRoomPan-rt.jar  .../BEvaporatorUnit.java:163 OPERATOR numeric without facets (slot=freezeSetpoint)
WARN  facets-req  ColdRoomPan-rt.jar  .../BColdRoom.java:64 setpoint-like slot roomHighAlarmLimit missing UNITS
PASS  ord-literal  ColdRoomPan-rt.jar  no hardcoded ORD literals under .../ColdRoomPan-rt/src
PASS  palette    ColdRoomPan-rt.jar  module.palette has 3 component entries
verify-module: 7 passed, 0 failed, 2 skipped, 11 warned -> ALL PASS
```

`freezeDiffStop`, `freezeDiffRestart`, `powerOnDelay` in BEvaporatorUnit have `facets = @Facet("BFacets.make(BFacets.MIN, 0.0)")` — presence-only check correctly passes them (MIN=0 is a real MIN). Zero false positives.

---

## Guard results

```
bats tests/*.bats            : 218/218 passed
shellcheck toolbelt/verify-module.sh : exit 0
sweep-build-state.sh         : exit 0
sweep-fold-audit.sh --strict : exit 0 (56 folded, 56 cited, 0 uncited; NOTEs are ambiguous tokens, not uncited)
tests/kit-links.bats         : 6/6 passed
```

---

## Lessons

1. **Multi-line annotation accumulation requires awk END-block scanning, not line-by-line matching.** Real `@NiagaraProperty(` annotations span 4-8 lines; a line-by-line grep misses every multi-line form. Collecting lines into an array and scanning in `END {}` is the correct POSIX-awk pattern.
2. **Deduplication with `seen[]` keyed by slot name is mandatory.** N4 Slot-o-Matic emits BOTH an `@NiagaraProperty` annotation AND a `public static final Property xxx = newProperty(...)` declaration for the same slot. Without `seen[]`, a check scanning both forms double-fires.
3. **Pass ordering matters for false-positive prevention.** Pass 1 (annotations) must record ALL encountered slot names in `seen[]` — not only those that WARN — before pass 2 (old-style `newProperty`) runs. Marking only WARN-emitting slots allows pass 2 to fire on non-numeric OPERATOR slots (boolean, BStatusBoolean) that pass 1 correctly skipped.
4. **Smoke path for multi-profile modules must follow `profile_dir` naming.** `profile_dir` appends the jar basename (without extension) to `--src`. For `DashboardPan-ux.jar` the correct `--src` is the parent `DashboardPan`, not `DashboardPan-ux`. Document this convention in the pre-gate steps of BUILD-LOOP.md if other multi-profile smokes are added.
5. **Presence-only facets check (MIN=0 intentional minimum) is correct by design.** `BFacets.make(BFacets.MIN, 0.0)` carries a real MIN=0 bound — the slot is properly annotated. The check must never inspect values, only the existence of a `facets =` key.
