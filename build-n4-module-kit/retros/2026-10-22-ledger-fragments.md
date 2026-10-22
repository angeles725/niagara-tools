<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · ledger-fragments

**Session**: niagara-research quality round 8, task T8i slice B, branch `r8/t8i-b-ledger-fragments` from origin/main c30a19a
**Delta count**: 2

## What happened
Slice A made `retros/INDEX.md` generated. The other hot spot was the `kit` block of `BUILD-STATE.md`: every kit
PR rewrote the single `last_change:` line, prepended `open_issues:` items at the same spot and appended a
session-log line at the same spot. With four sessions in parallel every kit PR conflicted there, and "keep both
sides" went wrong at least three times. This unit moves those three parts into one fragment file per change under
`ledger.d/` and generates them with a new Go tool, `go/cmd/gen-build-state` (wrapper
`toolbelt/gen-build-state.sh`, `--check` in CI and in `sweep-build-state.sh`). The 95 session-log lines and 76
open issues moved into `ledger.d/0000-archive-pre-2026-10-23.md`; the generator reproduces the origin/main
`BUILD-STATE.md` byte for byte from it. The pre-push gate counts a fragment as the `BUILD-STATE.md` update.
It also fixes the #290 advisory: `go/go-tool.sh` now fails closed when its staleness `find` fails.

## Evidence
- origin/main c30a19a kit block: one `last_change:` line, 76 `open_issues:` items, 95 log lines after
  `last_session:`, none of them in strict date order (a 2026-10-23 line sits before 2026-10-22 lines)
  `[ev: build-n4-module-kit/BUILD-STATE.md@c30a19a]`
- `gen-build-state --check` over origin/main's `BUILD-STATE.md` plus the archive fragment alone: fresh
  `[ev: tests GBS-real, go TestGoldenRealLedger]`
- Gate cases H13-H16 and sweep case GBS-sweep RED before the hook and sweep changes, GREEN after
  `[ev: tests/build-retro-sync.bats, tests/gen-build-state.bats]`

## Proposed kit deltas (applied in this unit)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The kit block's `last_change`, `open_issues` items and session log are generated from one `ledger.d/<date>-<slug>.md` fragment per change; a fragment counts as the BUILD-STATE update for every close exit, and a stale file fails CI, sweep and the gate. | `BUILD-LOOP.md` §7, `BUILD-STATE.md` How to read | `[ev: retro ledger-fragments Δ1]` |
| Δ2 | Generated regions are found by structure (the `last_change:` line, the items under `open_issues:`, the log lines after `last_session:`), not by marker comments, so the migration is byte-identical; a non-log line after `last_session:` is an error, never silently dropped. `BUILD-STATE.md` gets no `merge=union`: union would also duplicate hand-kept module fields without any check, so a conflict is resolved by taking either side of the generated lines and regenerating. | `BUILD-LOOP.md` §7 | `[ev: retro ledger-fragments Δ2]` |

## Lessons
- Find the regenerated part by its structure when you can: marker lines would have broken the byte-identity proof.
- A file that every change edits must be derived from files that each change adds; then resolving a conflict is a regeneration, not a judgement.
- `cmd | head -1` inside `$(...)` hides `cmd`'s exit status; a staleness check must take the status from the command itself.

---
**Status**: FOLDED — both deltas applied in this unit.
