<!-- review-status: folded -->
# 2026-09-07 · ColdRoomPan · freeze-stat-silent-surface

**Session**: PANCCADIA — surface the freeze-stat trip (was silent) as a per-evaporator status, mirroring defrost-active.
**Delta count**: 1

## What happened
The freeze-stat (coil anti-frost low-limit) trips and CLOSES the valve — inhibiting cooling — with NO status/alarm surface; the operator sees a room not cooling but not WHY. `lint-silent-protection` flagged it (`valveInhibited` forces `freezeTripped`, no surface in scope). Added a per-unit READONLY status slot `freezeActive`, exposed it through the DashboardPan facade + reader, and showed "Protección anti-hielo activa: Evap X" in the SPA proc-line, mirroring the existing defrost-active idiom.

## Evidence
- `lint-silent-protection` WARN at `BEvaporatorUnit.java:1521` BEFORE → **0 WARN** after [ev: lint-silent-protection.sh]
- `freezeActive` (BStatusBoolean TRANSIENT|SUMMARY|READONLY) published in `recomputeFreeze()`; `valveInhibited` refactored `if(freezeTripped) return true; return resistHand;` — behaviour-equivalent, valve still closes [ev: BEvaporatorUnit.java]
- facade `evapMFreezeActive` (3) + `DashboardReader.STATE_SLOTS` + SPA proc-line text; 64 pure tests green; `ColdRoomPan 2.1.2`, `DashboardPan 2.4.2` [ev: build.sh both]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Doctrine for clearing a `lint-silent-protection` WARN: MIRROR an existing "*Active" status idiom in the same module (here defrost-active) rather than inventing a new surface — a per-unit READONLY SUMMARY status slot + facade slot + the same proc-line string style. Document the "mirror the active-status surface" recipe as the standard fix. | types/logic.md §protection-surface; lint-silent-protection.sh header | [ev: freezeActive] |

## Lessons
- A protection that changes actuation MUST publish a status/reason surface; the control (valve close) working is not enough — silent inhibition reads as a mystery failure.
- Refactor `return A || B` into `if (B) return true; return A;` to attach the surface at the exact trigger without changing behaviour.
- Reuse the module's existing active-status pattern so the operator UI stays consistent (defrost-activo ↔ protección-activa).

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-freeze-stat-silent-surface.md | ColdRoomPan | 2026-09-07 | pending | 1 |`
