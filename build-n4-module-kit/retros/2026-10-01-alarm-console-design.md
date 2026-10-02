<!-- review-status: pending -->
# 2026-10-01 · kit · alarm-console-design

**Session**: PANCCADIA gap review — when is a fault a status flag and when is it a console alarm
**Delta count**: 3

## What happened
CompPan faults (no-amps, protection trips) are exposed as `BStatusBoolean` flags only. The Fase 2 task explicitly leaves
console alarms (`BAlarmSourceExt`) out of scope as a "possible follow-up", so nobody decided per fault whether it needs an
alarm class, a recipient, acknowledgement, or debounce. The kit tells the builder HOW to wire a console alarm (Pattern A/B
in the protection anatomy) and lints that a protection trip has SOME operator surface, but not WHETHER and HOW MUCH
(class, recipient, ack, nuisance control). The corpus block on the alarm framework is not cited in the kit.

## Evidence
- Flags only, alarms deferred: "No `BAlarmSourceExt` in any of these modules; alarms are BStatusBoolean flags." and "Out of scope: ... Niagara alarm-console records (BAlarmSourceExt) — possible follow-up." `[ev: Cliente/panccadia-leon odd/tasks/comppan-fase2-amps-alarms.md:15,22]`
- Fault surfaced only as `condenserNFault` BStatusBoolean, not on the facade or dashboard (same task file, problem section) `[ev: Cliente/panccadia-leon odd/tasks/comppan-fase2-amps-alarms.md]`
- Kit covers the HOW: R15.2 routes logic faults through `BAlarmSourceExt` `types/logic.md:240`; Protection anatomy Pattern A (child point + `BAlarmSourceExt`) `types/logic.md:257-258`; Pattern B passes an `alarmClass` to `AlarmSupport` `types/logic.md:268` `[ev: types/logic.md:240,257,268]`
- Kit has no decision guidance: grep for `alarm class|recipient|nuisance` over `types/*.md` and `BUILD-LOOP.md` hits only `types/logic.md:268` (constructor argument) and two `types/distribution.md` migration mentions `[ev: kit grep 2026-10-01]`
- `lint-silent-protection.sh` checks for the presence of a status slot, `BAlarmSourceExt` or a readable reason, so a latch with a console alarm but no recipient or class passes `[ev: toolbelt/lint-silent-protection.sh:1-10]` [INFER on the "no recipient" part; the lint does not model recipients]
- Corpus B34 "Alarm framework deep + .adb format" is cited in no kit file `[ev: corpus B34]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | New section "Flag vs console alarm": a decision table per fault (operator must act now -> console alarm; informational/diagnostic -> flag), alarm class choice, recipient/routing, acknowledgement policy, and debounce/nuisance control (delay on-time, latch vs auto-clear). | `types/logic.md` (new §) after § `Protection anatomy `[ev: corpus B827]`` | `[ev: corpus B34]` |
| Δ2 | Commissioning checklist row: each alarm source has an alarm class and at least one recipient, and a test alarm reaches the console and is acknowledged. | `BUILD-LOOP.md` § `6.b Commissioning-verify requirement (modules needing station rewiring)` | `[ev: corpus B34]` |
| Δ3 | Extend `lint-silent-protection.sh` with an advisory WARN for a latched fault whose only sink is a console alarm with no status/reason slot (no local surface when the console is unattended). | `toolbelt/lint-silent-protection.sh` header | `[ev: types/logic.md:246-268]` |

## Lessons
- A flag nobody watches is a silent failure; an alarm nobody can acknowledge is noise.
- "Possible follow-up" in a task file is a decision not taken; record who decides.
- The kit documents the wiring but should also document the judgment.
