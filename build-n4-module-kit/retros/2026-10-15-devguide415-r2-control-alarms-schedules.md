<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 10 -->
# 2026-10-15 · kit · devguide415-r2-control-alarms-schedules

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 2 mined the 4.15.6.20 Alarms, Histories, KitControl and Scheduling user guides (plus the source behind them) for cold-room/compressor modules; this retro carries corpus B1324, B1331 and B1335. Complements `2026-10-15-devguide415-runtime-api-corrections.md` (**rac** Δ6 alarm ext owns FAULT) and `…-horizons-authoring-security.md` (**has** Δ3 history `_cfg` split).
**Delta count**: 10

## What happened
The kit teaches alarm/history/timer mechanics but never the platform blocks our control modules could reuse, and one default is wrong for 4.15. Alarms: `ackRequired` defaults to toOffnormal|toFault|toNormal in 4.15.6.20 (the kit says "without toNormal", a 4.14 reading); `timeDelay` never debounces FAULT; algorithms skip disabled/stale points, so a stale probe never alarms; `alarmInhibit` is a linkable `BStatusBoolean` (defrost/compressor-off can suppress high-temp alarms); alarm texts are raw BFormat resolved against the alarm record. Histories: an ext ships disabled (`DEFAULT_ON_CLONE`), a literal name loses auto-numbering and faults when duplicated. KitControl: LeadLag rotates one output every 6 min with TRANSIENT state and no min on/off; SequenceLinear ships a 1 s delay (the guide says none); the only built-in anti-short-cycle is InterstartDelayControl; a new LoopPoint has zero gains. Scheduling: a TriggerSchedule gives one persisted, operator-editable entry per defrost and fires `triggerMissed` once on any recompute past the persisted `nextTrigger`; ColdRoomPan today derives defrosts from BooleanSchedule edges.

## Evidence
- `ackRequired` `[ev: code niagara-help/versions/4.15.6.20/source/alarm-rt/javax/baja/alarm/BAlarmClass.java:178]` vs `[ev: code types/logic.md:324]`; fault/no delay `[ev: code …/alarm/ext/BAlarmSourceExt.java:1634]` `[ev: niagara-help 4.15.6.20 guides-clean/Alarms/InhibitingAlarms-AA408864.txt:15 "Note: Time Delay does not affect alarms generated"]`; stale skip `[ev: code …/BAlarmSourceExt.java:1462]`; inhibit `[ev: code …/BAlarmSourceExt.java:326]`; history ext disabled `[ev: code …/history-rt/javax/baja/history/ext/BHistoryExt.java:281]` `[ev: corpus B1324]`
- LeadLag `[ev: code niagara-help/versions/4.15.6.20/source/kitControl-rt/com/tridium/kitControl/hvac/BLeadLagRuntime.java:307,385,411]`; Sequence `[ev: code …/hvac/BSequence.java:575]` vs guide `[ev: niagara-help 4.15.6.20 guides-clean/KitControl/kitControl-SequenceLinear.txt:38 "must pass between changes in outputs"]`; LoopPoint `[ev: code …/BLoopPoint.java:499,352]`; BooleanDelay `[ev: code …/timer/BBooleanDelay.java:535]` `[ev: corpus B1331]`
- TriggerSchedule `[ev: code niagara-help/versions/4.15.6.20/source/schedule-rt/javax/baja/schedule/BTriggerSchedule.java:274,524,527,540]`; ColdRoomPan schedule mode `[ev: code Cliente/panccadia-leon/Paccadia/ColdRoomPan/ColdRoomPan-rt/src/com/angeles/ColdRoomPan/BDefrostMode.java:16]` `[ev: corpus B1335]`
- Kit grep 2026-10-15 over `types/` for `alarmInhibit|ElapsedActiveTime|CommandFailure|LeadLag|Interstart|BooleanDelay|Tstat|TriggerSchedule|ScheduleSelector|triggerMissed` → 0 hits

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Make `ackRequired` version-qualified (4.15.6.20 = toOffnormal\|toFault\|toNormal); state that `timeDelay` debounces offnormal only and that disabled/stale points are never evaluated (add a `StatusAlarmExt` or module stale alarm). Lint: partial. | `types/logic.md:324-325` | `[ev: corpus B1324 §1324.1 rows 1,3,6]` |
| Δ2 | Add an alarm-ext chooser and inhibit recipe: `BooleanCommandFailureAlarmExt` (compressor fault), `ElapsedActiveTimeAlarmExt`+`DiscreteTotalizerExt` (run hours), `OutOfRange`/`FloatLimit` (temps), link `alarmInhibit` from defrost-active/compressor-off; write texts against the record (`%alarmData.presentValue%`). Lint: yes — WARN on `%parent.` in alarm text defaults. | `types/logic.md` § alarm decisions | `[ev: corpus B1324 rows 2,4,5]` |
| Δ3 | History rules: palette exts ship disabled (enable at commissioning or `setEnabled(true)`); keep `historyName` a script; set capacity; Min/MaxRollover for counters; never rename from the History space. Reinforces has Δ3. Lint: yes — flag a literal `historyName`. | `types/logic-authoring.md:536-545` | `[ev: corpus B1324 rows 8-10]` |
| Δ4 | Commissioning: size the file alarm DB (default 500, oldest overwritten). Lint: no. | `BUILD-LOOP.md` commissioning | `[ev: corpus B1324 row 7]` |
| Δ5 | Add "REUSE before you hand-roll": LeadLag = equal-wear selection only (raise `maxRuntime` far above 6 min; keep short-cycle protection in our code); Sequence for non-critical staging (1 s default delay, restages from 0 on restart); InterstartDelayControl for inrush + Min Active/Inactive. Lint: partial. | `types/logic.md` after `:333` | `[ev: corpus B1331 rows 1-3]` |
| Δ6 | Timers: BooleanDelay/OneShot/NumericDelay outputs are TRANSIENT (boot with In=true re-fires the on-delay; min-off cannot survive a restart); Counter persists. Lint: no. | same | `[ev: corpus B1331 rows 4-5]` |
| Δ7 | LoopPoint: a dropped loop has kP=kI=kD=bias=0 (outputs only bias) and runs on a 0.5 s time ticket, not on link changes. Lint: partial — flag kP=0. | `types/logic.md:332-333` | `[ev: corpus B1331 row 6]` |
| Δ8 | Tstat has no lockout (pair with BooleanDelay/Interstart); totalizer/alarm/history exts only on point-based blocks, alarm ext ordered below its totalizer. Lint: partial. | same | `[ev: corpus B1331 rows 7-8]` |
| Δ9 | Use a `TriggerSchedule` for fixed wall-clock actions (N defrosts/day): one persisted ticket, recomputed on start/clock change/edit; make the action idempotent; link `triggerMissed` to a late-defrost policy; reserve `Clock.schedule` for relative delays. Lint: partial — WARN on `Clock.schedule(this, BAbsTime…)` without `clockChanged`. | `types/logic.md` § watchdogs and timers | `[ev: corpus B1335 rows 1-4]` |
| Δ10 | Weekly schedules output `defaultOutput` when ineffective (Boolean false, others null = relinquish); specials by list order; cleanup deletes past one-time events; share one schedule via `ScheduleSelector` or fan-out, never one copy per room. Lint: no. | `types/logic-authoring.md:298` | `[ev: corpus B1335 rows 5-7]` |

## Lessons
- Before writing control logic, check whether a stock block already does it — and read its defaults and TRANSIENT flags, which decide restart behaviour.
- A default copied from a 4.14 decompile must carry its version; 4.15 changed `ackRequired`.
- Persisted schedule state (`nextTrigger`) is what makes "missed" detection possible; a hand-made timer loses it.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — Δ1 `types/logic.md:384`, Δ2 `:403`, Δ3 `types/logic-authoring.md:636`, Δ4 `BUILD-LOOP.md:296`, Δ5-Δ8 `types/logic.md:396-402`, Δ9 `types/logic-authoring.md:362`, Δ10 `types/logic-authoring.md:360`. INDEX row generated from the header: `| 2026-10-15-devguide415-r2-control-alarms-schedules.md | kit | 2026-10-15 | folded | 10 |`
