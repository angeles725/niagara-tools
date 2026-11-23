<!-- review-status: folded 2026-11-23 · kit 7934eca5 -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. Rows are marked APPLIED with their commit in the PR that lands them. -->
# Retro — lab-415 · 2026-11-15 · alarm tools and a kit-scaffolded module on a laboratory station

> Written by hand from the implementer's live runs of the same day: the first live use of a
> module scaffolded by the build kit (`KitBenchDash`, vendor `benchco`, self-signed), and the live
> proofs of the new history and alarm tools. "Observed" means read from a tool reply, a read-back,
> a spy page or the saved `config.bog`.

## Session summary

- Station: a laboratory station on Niagara 4.15.6.20 (tier B opt-in); the operator authorized every
  write; every scratch folder was removed and the station saved, 0 hits in `config.bog`.
- Kit module: the registry listed `KitBenchDash-rt` (type `KitBenchDash:KitBenchDash`, a `BComponent`
  with the doubles `setpoint` and `status`) and `KitBenchDash-ux` (`KitBenchDash:KitBenchDashServlet`).
  `n4_create_component` read `verified`; `n4_set_slot setpoint 21.5` read `verified`.
- Alarms: `n4_add_alarm` (out of range, boolean change of state, a 15 s time delay, an inhibited
  extension) read `verified`; the alarm database recorded offnormal and back-to-normal transitions with
  the configured texts; the delayed one alarmed about 15 s later and the inhibited one never did; the
  rollback of each add read `verified` and left no extension.

## Proposed kit deltas

| # | Proposed change | Target (file · section) | Evidence (batch_id / tool call / audit line) | Type | Priority |
|---|---|---|---|---|---|
| 1 | **APPLIED 2026-11-15 (commit `e9e95e37`, PR #678, live-proven on the lab station).  ** `n4_read_slots` and `n4_navigate` fill the frozen properties a type declares from the type contracts (`reg/loadTypes`), marked `default`, when the load omits them; when the contracts cannot be read they say so instead of answering an empty list | `go/internal/mcpn4/readtools` (readslots, navigate) | observed: a fresh `KitBenchDash` read `slots: []` although the class declares `setpoint` and `status`; `setpoint` showed only after it was written and `status` never showed (a BOX load omits a frozen property still at its default) | tool | high |
| 2 | **APPLIED 2026-11-15 (commit `e9e95e37`, PR #678, live-proven: the row-2 rollback reads verified).  ** The rollback read-back counts a frozen property that the load omits as its declared default, so restoring a default reads `verified` | `go/internal/mcpn4/writeops` (rollback read-back) | observed: the rollback of `n4_set_slot alarmInhibit true` on an alarm extension read `mismatch` with `restored: false` while the slot was back at `false` (omitted by the load because it is the default); same root cause as row 1 | tool | high |
| 3 | **APPLIED 2026-11-15 (commit `e9e95e37`, PR #678).  ** `n4_spy_page` accepts a `module:Type` segment under `sysManagers/registryManager/types/` only, and keeps refusing schemes and ORDs elsewhere | `go/internal/mcpn4/spy` (path rule) | observed: `sysManagers/registryManager/types/KitBenchDash:KitBenchDash` refused "it holds a scheme or an ORD"; the module page `…/modules/KitBenchDash-rt` read fine | tool | medium |
| 4 | **DECLINED 2026-11-23 (the refusal is right and undoing a scratch test is two `n4_rollback` calls newest first, each with its own verdict; a range rollback would blur which batch failed to restore)**  A range rollback (`n4_rollback` of several batches, newest first, one token) for a scratch test that created a component and then wrote under it | `go/internal/mcpn4/writeops` (rollback) | observed: the rollback of a create was refused because a later `n4_set_slot` batch wrote under it (the refusal is right); undoing the test took two rollbacks in order | tool | low |
| 5 | **APPLIED this pass (`n4_set_slot` `status` takes flag names, `fault`, `down,overridden`, `{alarm,unackedAlarm}`, `ok`, written whole as the hex bits; the hex form stays; `blackbox.TestSetSlotStatusTakesFlagNamesAndWritesThemAsTheHexBits`, `TestSetSlotStatusKeepsTakingAHexStringAndRefusesAnUnknownName`)**  `n4_set_slot` takes an optional `status` (flags such as `fault`, `down`, `overridden`) with `baja:StatusNumeric` / `baja:StatusBoolean`, written whole with the value, so a fault can be forced on a test point | `go/internal/mcpn4/writeops` (set slot) | observed: `fallback` with value `"20.0 {fault}"` refused "value must be a finite number"; `fallback/status` refused "Status slots are written whole"; for an out-of-range alarm the fault was then forced through the fault algorithm's own limits (`fault_high_limit` 100, point at 150: record `Fault` with the fault text), so this row is for other kinds (boolean, enum) and for testing status-driven logic | tool | low |
| 6 | **DEFERRED 2026-11-23 (needs a live station: how `alarmData.<key>` columns render, localized numbers such as `60,0` included, in a second select over the same records cannot be proven offline, and a wrong resolution would show invented text)**  `n4_read_alarms` returns `msg_text_resolved`, the message with every `%alarmData.<key>%` replaced from the record's `alarmData`, beside the raw pattern | `go/internal/mcpn4/readtools` (alarms) | observed: the database stores the raw pattern (`%alarmData.sourceName% HIGH %alarmData.presentValue% > %alarmData.highLimit%`) and `alarmData` holds `sourceName=N`, `presentValue=60,0`, `highLimit=50,0` | tool | medium |
| 7 | **APPLIED this pass (`n4_read_slots` `slots`, with `slots_not_found`; `readtools.TestASlotsListReturnsOnlyThoseSlotsInStationOrder`, `TestASlotsListNamesTheSlotsTheComponentDoesNotHave`, `TestWithoutASlotsListEverySlotIsStillRead`)**  `n4_read_slots` with a `slots` list returns only those slots | `go/internal/mcpn4/readtools` (readslots) | observed: `slots: ["alarmInhibit"]` on an alarm extension also returned `timeInCurrentState`, `faultAlgorithm`, `offnormalAlgorithm` and `status` | tool | low |
| 8 | **DEFERRED 2026-11-23 (needs a live station: the AlarmService actions `enableToOffnormal` / `disableToOffnormal` and the `alarm` channel keys `addNoteToAlarms` / `forceClear` were found by research and never invoked; each tool needs its argument encoding proven live first; the recipient link recipe is already in `README.md`)**  Integrator needs met on the way, not built: bulk enable or disable of alarm extensions (AlarmService `enableToOffnormal` / `disableToOffnormal` with a list of ORDs), alarm notes and force-clear through the BOX `alarm` channel, a recipient recipe (console, email) with its links, schedules, tags and relations | `skill/SKILL.md` · new tools | research for the alarm tools: AlarmService actions and the `alarm` channel keys `addNoteToAlarms`, `forceClear` exist; tags and NEQL are bench retro 2 row 11 | new | low |

## Already covered (dedupe)

- Alarm extension, class recipe, alarm read and acknowledge: bench retro 2 row 10 (this batch's PRs).
- The history type, capacity and record reads: bench retro 2 rows 7 and 8, client-a history retro rows 2 and 4.
