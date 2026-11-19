<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-19 · kit · mcpn4-copy-keeps-status

**Session**: a field JACE (4.14, client B's site); 28 schedule copies made with `n4_copy_component` to give each HVAC unit its own power and setpoint schedule.
**Delta count**: 2

## What happened
The source schedules had `in` and `defaultOutput` at {null}. Every copy landed with `in` = false / 0.0 {ok} and the NumericSchedule `defaultOutput` = 0.0 {ok}, so the copies' `out` was false / 0.0 {ok}, and the read-back still said `verified`. Linking them into the units' writables would have switched 14 units OFF and set 14 setpoints to 0 °C. A manual BQL read of `out` before the links caught it; the copies were fixed by hand with `n4_set_slot` status "40".

## Evidence
- `copy_strip.go` `stripTree` dropped every child declared readonly; the `status` child of a Status value is declared readonly, so it never travelled and the copy fell back to {ok}.
- The read-back ran the same strip on the copy, so it never compared status.
- Fix `9f7a7df2`: RED `TestACopyOfANullScheduleCarriesTheNullStatus` (both schedule kinds) and `TestACopyWhoseStatusDiffersFromTheSourceIsAMismatch`; `TestAPlainCopyStillPlansTheSameOps` guards plain copies.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-19. `n4_copy_component` carries a Status value's status when it is off its default (encoded like `n4_set_slot`), and the read-back compares value and status | `go/internal/mcpn4/writeops/copy_strip.go`, `copy_status.go`; registry description | `[ev: retro mcpn4-copy-keeps-status Δ1]` |
| Δ2 | OPEN. `n4_create_link` dry run states what the source commands the target to right now (source `out` value and status), and notes a link from an {ok} control source into a writable slot | `go/internal/mcpn4/writeops/link*.go` | `[ev: retro mcpn4-copy-keeps-status Δ2]` |

## Lessons
- A "verified" read-back that strips the same fields as the copy cannot see what the copy lost. The read-back must compare what matters in the field (here, status), not the copied body.
