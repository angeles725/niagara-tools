<!-- review-status: folded -->
# 2026-09-07 · ColdRoomPan · coldroompan-indefrost-resistance-recovery

**Session**: PANCCADIA León live commissioning (per-evap rearchitect, ColdRoomPan v2.1.x) on the JACE-9000 over oBIX. Bug found live; code fix owed. Companion to the kit retro `2026-09-07-live-commissioning-verification-gaps`.
**Delta count**: 2

## What happened

A `BEvaporatorUnit` can be left with `inDefrost == true` stranded (config changed mid-defrost, or a per-evap `BDefrostController` created/replaced while a room-level cycle had the unit in defrost, so `exitDefrost()` is never called for it). Once stranded, the unit is frozen in the defrost output pattern — **`resistanceOut` stuck ON, `valveOut`/`evapOut` OFF** — and there is **no recovery path**: every output-writing method returns early under `if (inDefrost) return`, `stopped()` clears `inDefrost` but deliberately does not rewrite `resistanceOut`, and the restart path (`computeEvapCall → applyRunCmd`) never writes `resistanceOut`. So a disable/enable does NOT clear it. Live on Cuarto1/EvaporatorUnit_1: `resistanceOut=true` with both DefrostControllers `defrostActive=false`. Live recovery was `forceDefrost` (air branch sets `resistanceOut=false` immediately), but the code must not depend on the operator noticing.

## Evidence
- `BEvaporatorUnit.java:1224` `applyHoaOutputs()` begins `if (inDefrost) return;` — the ONLY auto path that sets `resistanceOut=false` is gated by the flag. `[ev: BEvaporatorUnit.java:1224]`
- `:1125` `applyRunCmd()` and `:1174` `applyFanRunMode()` share the same `if (inDefrost) return` guard. `[ev: BEvaporatorUnit.java:1125]`
- `:1053-1056` `stopped()` clears `inDefrost` (`:1058`) but comments explicitly forbid writing `resistanceOut` there. `[ev: BEvaporatorUnit.java:1053]`
- `computeEvapCall()` calls only `applyRunCmd()` (never `applyHoaOutputs`) — restart does not clear `resistanceOut`. `[ev: BEvaporatorUnit.java:1288]`
- Live: `Programacion/ColdRoom_1/EvaporatorUnit_1` `resistanceOut=true`, `valveOut=false`, `evapOut=false`, both controllers `defrostActive=false`. Cleared only by `forceDefrost`. `[ev: live oBIX PANCCADIA JACE 2026-09-07]` `[ev: engram #8443]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | ColdRoomPan code fix: on `started()`/`atSteadyState()` (unit enable), when `Sys.atSteadyState()` and NOT genuinely in an owned defrost cycle, force `resistanceOut=false` before applying the cooling call — an unconditional reset so a stranded `inDefrost` cannot leave the heater energized. (Equivalently: `computeEvapCall`/enable path calls `applyHoaOutputs`, or resets `inDefrost=false` + `resistanceOut=false` when no controller owns the unit.) | ColdRoomPan `BEvaporatorUnit.java` `started()`/`computeEvapCall` | `[ev: engram #8443]` |
| Δ2 | Add `lint-recovery-path` (kit): FAIL when a protection/heat output is written to a safe/off value ONLY inside methods gated by a transient mode flag, with no unconditional reset on `started()`/enable. | `toolbelt/` + `BUILD-LOOP.md` § step 5 | `[ev: BEvaporatorUnit.java:1224]` |

## Lessons
- **A transient mode flag that gates every output writer must never own the only path that clears a protection output.** Provide an unconditional reset on start/enable.
- **`stopped()` clearing a flag is not recovery** if the restart path never re-drives the outputs — the field device holds the last value, so the heater stays energized.
- **`forceDefrost` is a live workaround, not the fix.** The module must self-recover so no operator vigilance is required. Ties to kit retro `live-commissioning-verification-gaps` Δ5.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-coldroompan-indefrost-resistance-recovery.md | ColdRoomPan | 2026-09-07 | pending | 2 |`
