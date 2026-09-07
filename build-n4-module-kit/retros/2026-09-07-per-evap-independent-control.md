<!-- review-status: pending -->
# 2026-09-07 · ColdRoomPan · per-evap-independent-control

**Session**: PANCCADIA per-evaporator rearchitect — PR1a/b/c chained slices, build+verify each.
**Delta count**: 2

## What happened
Client change: evaporators must no longer be controlled by the ZONE setpoint — each evaporator controls on its OWN coil sensor + setpoint, fully independent, with INDEPENDENT defrost. Today control was zone-temp vs one room setpoint (evaporators staged together) and defrost was SERIALIZED (one BDefrostController per room, token+FIFO). Sliced into PR1a (additive per-unit control slots + dormant `computeEvapCall`), PR1b (activate the driver, `BColdRoom` becomes OR-aggregator, `roomMinSetpoint`), PR1c (defrost topology: one controller per evaporator).

## Evidence
- `computeEvapCall()` reuses pure `ColdRoomControl.decideCall` verbatim against `coilTemp` vs per-unit setpoint — zero new control math [ev: BEvaporatorUnit.java]
- `BColdRoom.execute()` = OR of units' `runCmd`, `coolingSince/idleSince` kept; `call1/call2`/staging/`driveUnit` retired [ev: BColdRoom.java]
- PR1c: `BDefrostController.units()` → `Collections.singletonList(getParent())`; every edge-case method (beginDefrost/terminateCurrent/doIntervalExpired) UNCHANGED [ev: BDefrostController.java]
- 64 pure tests green across 3 slices; `lint-delays` 4 PASS, `lint-timers` 2 PASS each build; `Paccadia 2.0.7 → 2.1.0` [ev: run-pure-test + build.sh]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Doctrine "re-home, don't rewrite": to move a room-scoped controller to per-unit, change ONLY the unit-resolver (`units()`) and re-parent per unit; reuse the controller body so every previously-fixed edge case (power-on hold, stale-ticket, HOA-OFF, time<=0) survives by construction. | types/logic.md §topology | [ev: BDefrostController.java] |
| 2 | Slice guard for a control-switch change: keep the ADDITIVE dormant slots (build-green, behaviour-unchanged) as PR-a, the actuation SWITCH as PR-b, the topology as PR-c — so a regression is isolated to one slice. | BUILD-LOOP.md §slicing | [ev: PR1a/b/c] |

## Lessons
- A pure hysteresis fn with no room/unit concept moves room→unit for free (reused verbatim).
- Keep room-level slots (setpoint/diffs) PRESENT-BUT-INERT so the deployed bog loads without breaking existing links.
- Station-level per-unit + independent-defrost behaviour is harness-only (a BComponent can't instantiate in WSL JUnit) — pure seams stay plain-JUnit.
- `build.sh` needs the GROUP dir as module-root + `niagara_home` as arg 3.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-per-evap-independent-control.md | ColdRoomPan | 2026-09-07 | pending | 2 |`
