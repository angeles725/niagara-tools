<!-- review-status: folded -->
# 2026-09-07 · ColdRoomPan · air-defrost-off-cycle

**Session**: PANCCADIA — add off-cycle (air) defrost for evaporators without resistance heaters.
**Delta count**: 1

## What happened
Cuartos 1/2/4 evaporators have NO resistance; their defrost is OFF-CYCLE / AIR: close the liquid valve, KEEP THE FAN RUNNING so room air melts the frost, no resistance, terminate by duration. The module had ONE defrost style (`enterDefrost`: valve off + fan off + resistance on). For a no-resistance unit it only "worked" because `resistanceOut` was left unwired AND it wrongly STOPPED the fan. Added an `airDefrost` per-unit flag branching `enterDefrost`.

## Evidence
- `enterDefrost()`: always close valve; `if (getAirDefrost())` → `evapOut=true` (fan ON) + `resistanceOut=false` (no resistance), skip the `defrostFanOffDelay` ticket; else path byte-for-byte unchanged (Cuarto 3 resistance defrost) [ev: BEvaporatorUnit.java enterDefrost]
- `cancelRunTickets()` preserved in the air branch → `powerOnTicket` survives (power-on-hold invariant) [ev: BEvaporatorUnit.java]
- extra guard: `changed()` mid-defrost resistanceMode gated with `!getAirDefrost()` [ev: BEvaporatorUnit.java]
- 64 pure tests green; `2.1.1`; lexicon `airDefrost=Deshielo por aire` [ev: run-pure-test + build.sh]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Anti-pattern lint/checklist item: a protection/defrost path that energizes an output (`setBool(xOut,true)`) for a unit whose hardware may be absent should be GATED by a per-unit capability flag (`hasX`/`airDefrost`) — never rely on "the output is unwired at commissioning". Flag an unconditional `resistanceOut=true` reachable by a `hasDefrost`-only gate. | toolbelt/lint-silent-protection.sh or a new capability-gate lint; types/logic.md | [ev: BEvaporatorUnit enterDefrost] |

## Lessons
- "No resistance" is not just an unwired output: an off-cycle/air defrost must KEEP THE FAN ON (room air melts the frost) — stopping the fan is wrong for it.
- Branch the entry sequence on a per-unit capability flag; keep the existing (resistance) branch untouched to protect Cuarto 3.
- The safe minimal edit is an `if (flag){...}` before the existing block — no else-restructuring, no moved code.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-air-defrost-off-cycle.md | ColdRoomPan | 2026-09-07 | pending | 1 |`
