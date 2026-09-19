<!-- review-status: folded -->
# 2026-09-19 · kit · logic-authoring-spis

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU expanding logic-authoring.md with exemplar SPIs + anatomy patterns.
**Delta count**: 8

## What happened
logic-authoring.md lacked several reusable SPIs found across real modules (program, email, weather,
backup) plus two code-anatomy patterns. Cluster anat-G6/G7 + exemplars C3/C4/C6/C7/C7b/C8.

## Evidence
- Verified against organized/ (alarm, program, email, weather, backup, control). `[ev: code BLocalAlarmResolver.java]` `[ev: code BWeatherService.java]` `[ev: code BFoxBackupJob.java]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | BSingleton+@NiagaraSingleton+@AgentOn general agent | `types/logic-authoring.md` | `[ev: code BLocalAlarmResolver.java]` |
| Δ2 | lifecycle guard contract (super order + isRunning + try/catch) | `types/logic-authoring.md` | `[ev: code BColdRoom.java]` |
| Δ3 | BProgram/BRobotCode scripting SPI | `types/logic-authoring.md` | `[ev: corpus program]` |
| Δ4 | BBatchRoutine mass-edit SPI | `types/logic-authoring.md` | `[ev: code BRenameBatchRoutine.java]` |
| Δ5 | BEmailService.send(BEmail) | `types/logic-authoring.md` | `[ev: code BEmailService.java]` |
| Δ6 | BIRestrictedComponent placement guard | `types/logic-authoring.md` | `[ev: code BEmailService.java]` |
| Δ7 | provider-in-service (BFolder+CoalesceQueue+Worker+tick) | `types/logic-authoring.md` | `[ev: code BWeatherService.java]` |
| Δ8 | BSimpleJob + Fox file-channel streaming | `types/logic-authoring.md` | `[ev: code BFoxBackupJob.java]` |

## Lessons
- @AgentOn generalizes beyond ORD schemes: any BIAgent singleton the framework finds by target type.
- The lifecycle guard contract (super-first, isRunning, try/catch; stopped cancels then super-last) is one rule, not per-method lore.

---
**Status**: FOLDED into `types/logic-authoring.md` (2026-09-19).
