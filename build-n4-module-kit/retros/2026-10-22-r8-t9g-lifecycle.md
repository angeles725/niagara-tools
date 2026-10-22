<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · r8-t9g-lifecycle

**Session**: niagara-research quality round 8, task T9g unit 3 (N4 developer course recordings, digest rows D1-12, D1-14..D1-18, D2-11..D2-13; research blocks B1416, B1417; plus the `pollEnabled` wording owed by T9e), branch `docs/r8-t9g-lifecycle` from origin/main 0678083
**Delta count**: 2

## What happened
Folding the spoken lifecycle lessons into `types/logic-authoring.md` showed the kit's own callback contract was
wrong: it said `stopped()`/`started()` fire on disable/enable. The engine calls them only on running-state changes;
services get `disabled()`/`enabled()` from `updateStatus()`, and a plain component sees `enabled` only in `changed()`.
The recordings also held a guess the code overturns: the instructor thought links activate after `started()`, but
`EngineManager.start` activates (and propagates) a component's links first.

## Evidence
- `com/tridium/sys/engine/EngineManager.java:74-90` (activateLinks → activateRelations → started → children → descendantsStarted); `javax/baja/sys/BLink.java:168,214-215` `[ev: corpus B1416 §1416.1]`
- `javax/baja/sys/BComponent.java:353-361` (stopped = running state false), `javax/baja/sys/BAbstractService.java:234-245,466,474,530-540` `[ev: corpus B1416 §1416.2]`
- `com/tridium/sys/station/Station.java:324` (stationStarted once, at bootstrap) `[ev: corpus B1416 §1416.3]`
- `javax/baja/driver/util/BAbstractPollService.java:26` (`pollEnabled` on the poll service) `[ev: corpus B1337 §1337.2 C1]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A lifecycle rule states which engine call fires the callback and cites it (`EngineManager`, the `BComponent` javadoc); "fires on enable/disable" style rules without a cite are re-checked before they are kept | `types/logic-authoring.md` § Lifecycle callback contract | `[ev: corpus B1416 §1416.2]` |
| Δ2 | A recorded instructor answer that is hedged ("I'm not 100% sure") is checked against the code before it becomes kit text; the block that reads the code wins | `types/logic-authoring.md` § Lifecycle guard contract | `[ev: corpus B1416 §1416.1]` |

## Lessons
- Disable is a status change, not a lifecycle transition.
- A course recording is evidence of what is taught, not of what the engine does.

---
**Status**: FOLDED (T9g unit 3, same unit) — Δ1 `types/logic-authoring.md` § Lifecycle callback contract, Δ2 § Lifecycle guard contract.
