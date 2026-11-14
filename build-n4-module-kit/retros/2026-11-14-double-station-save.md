<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-14 · kit · double-station-save

**Session**: operator read of a client A controller station log (4.15.3.28) after a day of dashboard edits.
**Delta count**: 3

## What happened
One operator change on a client A controller produced two station saves a few seconds apart, every time. The
station carries two modules built with this kit, a control module and its dashboard module, and each one requested
its own debounced save for the same change. Both modules follow the kit rule to the letter: `types/logic.md`
§ Restart & persistence tells an author to give each component "ONE pending save ticket (`ConfigSaveScheduler`)",
and says nothing about a station that holds more than one such component. The debounce coalesces a burst inside
one owner and nothing coalesces across owners, so the number of `config.bog` writes per change grows with the
number of modules that see it. On a controller that is duplicated flash wear and a second 2 s save for no gain.

## Evidence
- Station log, 2026-11-13, seven operator changes between 08:55 and 14:41. Each shows the pair
  `[<control module>] config-save: requested station save (debounced engineering-config change)` and
  `[<dashboard module>] ConfigSaveScheduler: debounced station save requested` within one second. Five of
  the seven show `Saving station...` twice and `Saved .../config.bog` twice, 2 s apart (08:58:03 and 08:58:05,
  08:59:48 and 08:59:50, 09:09:06 and 09:09:08, 13:43:13 and 13:43:15, 14:41:40 and 14:41:42); the other two
  collapsed into one save only because both requests landed before the first save started.
  `[ev: client A station log 2026-11-13, pasted by the operator, not vendored]`
- The scheduler is one class copied verbatim into each module, one instance per owning facade, each calling
  `Sys.getStation().save()` on its own ticket: `<control module>-rt/.../ConfigSaveScheduler.java:72`,
  `<dashboard module>-rt/.../ConfigSaveScheduler.java:80` (its header says it mirrors the scheduler of the
  two other client modules verbatim, "One instance per owning facade"), instantiated in
  `BCompressorControl.java:5110`, `BCompressorPanel.java:2515` and `BRoomPanel.java:3572`.
  `[ev: client A module tree 29a5808]`
- The kit rule that produced it: `types/logic.md` § Restart & persistence, first bullet, "plus ONE pending save
  ticket (`ConfigSaveScheduler`: a burst of writes re-arms or coalesces into a single save, never one save per
  write)". `[ev: types/logic.md § Restart & persistence, first bullet]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The debounced-save rule is per STATION, not per component: one operator change yields at most one station save however many modules observe it. Doctrine names the mechanism: the component that owns the persisted slot requests the save; a module that only forwards or mirrors an operator write (a dashboard facade) requests none for that write; where two owners can both fire, the ticket skips its save when a station save started after the change it is covering (compare against the last save time) | `types/logic.md` § Restart & persistence · `types/dashboard.md` (dashboard facade writes) | `[ev: client A station log 2026-11-13, pasted by the operator, not vendored]` |
| Δ2 | The scheduler stops being copy-pasted per module: the kit ships one reference `ConfigSaveScheduler` with the cross-owner guard of Δ1 and a pure policy test for it (two owners, one change, one save; a later change still saves), and the scaffold/checklist point at it instead of "mirror verbatim" | `templates/` (reference class + test) · `CHECKLIST.md` persistence row | `[ev: client A module tree 29a5808]` |
| Δ3 | The live check counts saves: after one operator config write, the station log shows exactly one `Saved .../config.bog` inside the debounce window plus save time; two is a FAIL row. The verify gate also reports, per client tree, how many classes call `Sys.getStation().save()`, so a second requester is visible before deploy | `BUILD-LOOP.md` § 6.b Commissioning-verify requirement · `toolbelt/verify-module.sh` (Go `internal/verifymodule`) | `[ev: client A station log 2026-11-13, pasted by the operator, not vendored]` |

## Lessons
- A rule written for one component must say what happens when a station holds several; "one ticket" per owner is
  N saves per change.
- A debounce only coalesces what goes through the same instance. Station-wide effects (save, restart, a bus scan)
  need a station-wide guard.
- "Mirror this class verbatim" in three modules is three copies of the same blind spot; a shared reference with a
  test is cheaper than the third copy.
- A green build and a working feature do not show duplicated writes. Only the station log does, so the live check
  must read it.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-14-double-station-save.md | kit | 2026-11-14 | pending | 3 |`
