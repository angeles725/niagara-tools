<!-- review-status: pending -->
# 2026-09-06 · kit · campaign10-ext-writable-per-slot

**Session**: C10 PR2 — S22 lint-ext-writable-shape per-slot writing-action exemption
**Delta count**: 4

## What happened
The C9 `lint-ext-writable-shape.sh` exempted a complex OPERATOR slot if the class had ANY
`@NiagaraAction` (class-level `has_action` flag). `BCompressorControl.faultReset`
(BStatusBoolean SUMMARY|OPERATOR at :381) was silently exempted by the unrelated HIDDEN
actions `tick`/`powerOnExpired`/`ackAlarm`, none of which write `faultReset`. The only
write (`setFaultReset(false)` at :2025) is inside `execute()` — excluded by construction
from the per-slot body scan. The fix replaces `has_action` with a per-slot body-scan
restricted to `do<Action>()` method bodies (resolved via the NRE `action x → doX()`
convention), plus a name-pattern fallback for `set<X>/apply<X>/<x>Cmd` action names
(B822 doctrine; EW3 positive pin). The section-D method-boundary parser from
`lint-silent-protection.sh` is ported with the `brace_depth>=2` guard (D1b) and an
additional fix for one-liner method bodies (`max_d` peak-depth tracking).

**CONTRACT CHANGE**: EW10 CompPan-rt re-pins from 0 to 1 WARN — subject `faultReset`
by NAME (lint anchors `@NiagaraProperty(` open at BCompressorControl.java:381, not the
generated `Property faultReset` at :1612). This is a false-negative fix, not a regression.
[ev: corpus B831 §831.2]

## Evidence
- `[ev: client BCompressorControl.java:381-385,:435-444,:2025 @ ff1b659]` — faultReset annotation, actions, and execute()-only write
- `[ev: qa/c10-ext-writable-per-slot 954ebd7]` — RED tip: EW-s22-neg, EW-s22-neg2, EW10 re-pin
- `[ev: bats GREEN 14/14 — EW-s22-pos clean, EW-s22-neg WARN, EW-s22-neg2 WARN, EW10 1 WARN]`
- `[ev: mutation 1: drop do_methods → EW-s22-pos flips to WARN]`
- `[ev: mutation 2: class-level has_action → EW-s22-neg2 flips to CLEAN]`
- `[ev: real-tree ff1b659: CompPan-rt 1 WARN faultReset :381, ColdRoomPan-rt 0, DashboardPan-rt 1 setpoint, DashboardPan-ux 0]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Document `max_d` peak-depth trick for one-liner methods in section-D parser shape | `METHODOLOGY.md` §parser | `[ev: EW-s22-pos one-liner fix]` |
| 2 | Add B831-G1 note: action `x` body writes Y ≠ X — must NOT exempt X | `types/logic-authoring.md` §ext-writable | `[ev: B831-G1]` |
| 3 | EW10 re-pin to 1 (faultReset) in campaign-10 contract change note | `BUILD-LOOP.md` §S22 | `[ev: D2c; D2d]` |
| 4 | Retro: the section-D parser needs max_d tracking for one-liner do<Action> bodies | `retros/INDEX.md` | `[ev: this retro]` |

## Lessons
- A class-level "has any action → exempt all slots" heuristic creates a false-negative gap whenever the action is unrelated to the slot (e.g. HIDDEN tick/powerOnExpired/ackAlarm in CompPan). Per-slot body-scan is mandatory.
- The section-D method-boundary parser tracks brace depth AFTER the full line; this misses one-liner `{ body }` methods. Tracking the per-line peak depth (`max_d`) and using it for `m_dep` fixes this.
- The do<Action>() scope filter is load-bearing: without it, the slotomatic-generated `setFaultReset()` inside `execute()` would exempt faultReset — exactly the false negative S22 was designed to catch.
- B831-G1: a `doAckAlarm` body that writes `alarmAck` must NOT exempt `faultReset`; the write-target must match the OPERATOR slot being checked, not any slot.
- EW10 contract changes (0→1) require a QA RED re-cut; without re-reading the RED at apply time (K13), a worker shipping "0 WARN" would pass the old pin and miss the real defect.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign10-ext-writable-per-slot.md | kit | 2026-09-06 | pending | 0 |`
