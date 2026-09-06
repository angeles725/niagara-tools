<!-- review-status: pending -->
# 2026-09-06 · kit · campaign10-silent-protection-pattern-b

**Session**: Campaign 10 PR3 — S23 lint-silent-protection Pattern B surface (BIAlarmSource/AlarmSupport adapter→pure follow)
**Delta count**: 3

## What happened

[ev: corpus B831 §831.3] The CP-1 low-suction shed in `CompressorControl.java:294` was flagged as a silent trip even after PR9 wired the alarm via the BIAlarmSource/newOffnormalAlarm pattern in `BCompressorControl`. The lint's surface criterion was file-level only — it checked `BAlarmSourceExt`/`BAlarmRecord` inside the SAME file as the trip. A pure class `C` and its adapter `BC` are in separate files, so the adapter's Pattern B surface was invisible to the lint when evaluating the trip in `C`. The fix adds a dir-wide Pass 0b `ALARM_CLASSES` index (class name keyed), then extends the surface criterion with an adapter→pure follow: `this_class ∈ ALARM_CLASSES` OR `"B" this_class ∈ ALARM_CLASSES`.

## Evidence

- `[ev: corpus B831 §831.3]` — architectural boundary: main awk ran per-file; surface criterion was file-level flag; no cross-file class-name follow existed
- `[ev: client CompressorControl.java:294 @ ff1b659]` — trip site in pure class
- `[ev: client BCompressorControl.java:447 (implements BIAlarmSource), :1882 (new AlarmSupport(), :2093 (newOffnormalAlarm) @ ff1b659]` — Pattern B surface in adapter
- `[ev: bats tests/lint-silent-protection.bats f981754]` — RED: S23-pos FAIL + SP-smoke CompPan-rt 1; GREEN after fix: 12/12 pass
- `[ev: real-tree smoke ff1b659]` — BEFORE: CompPan-rt 1 WARN (:294); AFTER: CompPan-rt 0 WARN (:294 absent); ColdRoomPan-rt 0 (Pattern A still recognised); DashboardPan 0
- `[ev: mutations]` — drop B<Pure> follow → :294 WARNs again; relax Pattern-B AND to OR → implements-only adapter wrongly clears trip

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Pass 0b ALARM_CLASSES index (Pattern A + Pattern B) | `toolbelt/lint-silent-protection.sh` Pass 0b | `[ev: D3b]` |
| 2 | Surface criterion extended with `"B"+this_class` adapter→pure follow | `toolbelt/lint-silent-protection.sh` section F | `[ev: D3c]` |
| 3 | D1b brace_depth >= 2 guard applied to section-D parser | `toolbelt/lint-silent-protection.sh` section D | `[ev: D1b]` |

## Lessons

- [ev: B831 §831.3] A file-level surface flag cannot cross the pure/adapter boundary: when the trip lives in `CompressorControl` and the surface is in `BCompressorControl`, the two-file separation is architectural, not accidental — the fix must cross that boundary deliberately via a dir-wide class-name index.
- [ev: D3b] Pattern B's `AND` requirement (implements BIAlarmSource AND newOffnormalAlarm/AlarmSupport) is load-bearing: relaxing it to OR exempts any class that merely imports or javadoc-mentions `BIAlarmSource` without actually raising an alarm.
- [ev: D1b] The `brace_depth >= 2` guard in section D (method-boundary parser) is a prerequisite for any caller of the parser: without it, a class body opening at depth 1 can be misidentified as a method when Case B's backward scan matches `defaultValue = "new BAlarmRecord()"`.
- [ev: K13] The RED commit is the contract: cherry-pick it as commit 1 and never merge it — a merge commit in a ff-only kit repo breaks the linear history required by the delivery contract.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign10-silent-protection-pattern-b.md | kit | 2026-09-06 | pending | 3 |`
