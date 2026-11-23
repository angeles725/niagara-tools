<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-11-19 · kit · mcpn4-link-commanded-value

**Session**: follow-up of retro mcpn4-copy-keeps-status Δ2 (delta P2 (c)); no station contacted, fake BOX station only.
**Delta count**: 1

## What happened
A link drives its source slot's current value into the target the moment it is made. The `n4_create_link` dry run showed only the checkLinks op, so 28 schedule copies at false / 0.0 {ok} looked like any other link; linking them into 14 units' writables would have switched them OFF and set setpoints to 0 °C. Only a manual BQL read of `out` caught it.

## Evidence
- RED: `write_link_commanded_test.go` (`TestAnOkScheduleIntoAWritableIn16IsNamedInTheDryRun`, `TestANullSourceShowsItsValueAndNoNote`, `TestAnOkSourceIntoANonWritableHasNoNote`, `TestABatchStatesEachEntrysSource`) failed with `link_sources = <nil>`; `hash_preview_test.go` failed to build (no `Planned.Preview`).
- Guards: `TestPreviewNeverChangesThePlanHash` and `TestASourceThatMovesAfterTheDryRunStillExecutesWithItsToken` both fail when the preview is hashed (mutation run, reverted); `TestAPlainLinksHashedPlanIsUnchanged` pins ops, inverse and notes.
- Design: the value is volatile (a PID or sensor `out` moves every second), so it cannot join the hashed notes without breaking execute; it rides `plan.Planned.Preview`, shown in the dry run like `link_input_values` and never hashed.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-19. `n4_create_link` dry run lists `link_sources` {value, status} per link outside plan_hash, and a note for an {ok} source into a writable's in1..in16 / fallback | `go/internal/mcpn4/writeops/link_source.go`, `link.go`, `link_batch.go`; `plan/plan.go` Preview; registry description | `[ev: retro mcpn4-link-commanded-value Δ1]` |

## Lessons
- Volatile state the operator must see belongs next to the plan, not inside the hash: a token bound to a moving value never executes.
