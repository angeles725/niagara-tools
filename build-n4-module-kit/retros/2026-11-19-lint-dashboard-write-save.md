<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-19 · kit · lint-dashboard-write-save

**Session**: issue #821, from a field dashboard retro (client B, N4 4.14): a `-ux` dashboard wrote commands, setpoints and schedules through three releases without a station save.
**Delta count**: 4

## What happened
The dashboard's writes changed the running station only. On a restart the Modbus driver re-sent the stale saved fallback (`writeOnStart`) and switched an HVAC unit ON with nobody in the building. BUILD-LOOP §6.b item 6 already asked for persisted operator state to survive the first restart, but as prose: nothing in the gate checked it. The fix (a debounced `BStation.save` scheduled by the servlet component) then hit `lint-delays`, which read a literal floor only in the second argument of `Math.max`.

## Evidence
- Real tree, read-only: the module at the last commit before its first save path gives 9 FAIL rows (`invoke(..., cx)`, `set(..., cx)`, `day.add(..., cx)` over two lines, `remove(..., cx)`, `addSpecialEvent(`); the shipped fix (`Clock.schedule(station, BRelTime.make(Math.max(delay, 1L)), BStation.save, null)`) is clean.
- RED: `go test ./internal/lint/rules/dashboardsave/` failed `truth-write-no-save-1`, `truth-save-not-code-1` and `empty-root-1` on a stub rule; `go test ./internal/lint/rules/delays/` failed `truth-max-floor-first-arg-1` with `zero-floor  cannot determine Math.max floor in: BRelTime.make(Math.max(1L, delayMs))`.
- A sweep over the other client `-ux` modules (read-only): two current modules write with the request Context and save nowhere in the module (a door-pulse action module and a dashboard that writes writables); each needs a save or a `persistence:` declaration, or its next report FAILs.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-19. Go rule `lint-dashboard-write-save` (module root): FAIL every write line of a `-ux` part (`.set(`/`.add(`/`.remove(`/`.invoke(` whose last argument is the request Context/WebOp, `addSpecialEvent(`) when the module requests no station save and its MODULE-MAP declares no `persistence:` line; BUILD-LOOP §6.b item 6 cross-links it | `go/internal/lint/rules/dashboardsave`, `BUILD-LOOP.md` §6.b item 6 | `[ev: retro lint-dashboard-write-save Δ1]` |
| Δ2 | APPLIED 2026-11-19. `types/dashboard.md` § Write path: debounced save exemplar (ticket on the servlet component, cancelled in `stopped()`, pure delay policy, `Math.max(delay, 1L)` at the call, never fail the write) and the `persistence:` escape | `types/dashboard.md` § Write path | `[ev: retro lint-dashboard-write-save Δ2]` |
| Δ3 | APPLIED 2026-11-19. `lint-delays` reads a bare literal floor in either argument of `Math.max` (inline, local binding, helper body) | `go/internal/lint/rules/delays` | `[ev: retro lint-dashboard-write-save Δ3]` |
| Δ4 | APPLIED 2026-11-19. Wired: `dashboardsave.Spec()` registered in `n4lint`, a `perModule` member of report-module (SKIP for a module with no `-ux` part), `toolbelt/lint-dashboard-write-save.sh` with its lint-index header and pins LDWS1-LDWS4 (`tests/lint-dashboard-write-save.bats`), and the `lint-delays.sh` header plus LD28/LD29 for both `Math.max` orders | `go/internal/lint/rules/rules.go`, `go/internal/reportmodule/members.go`, `toolbelt/`, `tests/` | `[ev: retro lint-dashboard-write-save Δ4]` |

## Lessons
- A rule that lives only in a checklist does not stop a defect; the gate does.
- A save is part of every operator write on a live station. When it is missing the failure shows only on the next restart, and on site it looks like equipment switching on by itself.
