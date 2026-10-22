<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 9 -->
# 2026-10-15 · kit · devguide415-r2-runtime-contracts-formats

**Session**: research-sdd focus `dev-capabilities-kit-gaps` round 2 (niagara-research B1316–B1343) closed the round-1 static child gaps from N4.15.6.20 source; this retro carries corpus B1316 (engine contracts), B1317 (Command keys, nav files, BFormat), B1319 (kitControl status) and B1336 (Expr null). Extends `2026-10-15-devguide415-runtime-api-corrections.md` (**rac** Δ5, Δ9, Δ10) and `…-nav-subscriptions-tags-ords.md` (**nsto** Δ5); confirms `2026-10-15-n4-course-threading-jobs-doctrine.md` (**tjd**) Δ3.
**Delta count**: 9

## What happened
Round 1 left nine child gaps that only source could settle. Reading them closed four kit errors and three silent fail-opens. (1) `BComponent.post()` coalesces FIRST-wins in the engine `ActionQueue` (a second post of the same action is dropped with its argument), while a `CoalesceQueue` of `Invocation`s is LAST-wins, and queued ASYNC actions run with a NULL Context. (2) A null Context, or one with no user, makes `BComponent.getPermissions` return `BPermissions.all`, so `OrdTarget.canRead/canWrite/canInvoke` pass — a servlet that resolves with `null` before checking permissions has no gate. (3) `add(name, BLink)`, `linkTo()` and `makeLink()` never run `checkLink`/`doCheckLink`; only tools do, and the slot map only checks user permissions (skipped for superusers and null Contexts). (4) `Command`/`ToggleCommand` read exactly `keyBase.label/.icon/.accelerator/.description`. (5) Nothing scans `rc/nav` or `stations/*/nav`; nav files load only through explicit ORDs (`BUser.navFile`, the mobile nav file). (6) BFormat's effective order is slot → six reflect steps → `BIFormatPropertyHandler`; `time()`/`user()` need `()` and take no argument; `substring` has three forms on the chained value; `?` starts a fallback script. (7) kitControl Select/Switch default `propagateFlags` to `nullStatus` and Logic/Math to `ok`, so a faulted probe produces an OK-status output, and an invalid selector HOLDS the last value. (8) `BHysteresis`/`BLimiter` are 4.15-only and the hysteresis latch is not persisted.

## Evidence
- First-wins `[ev: code organized/baja/baja/vineflower/com/tridium/sys/engine/ActionQueue.java:28,85]`; null Context run `[ev: code …/engine/EngineManager.java:310]`; last-wins `[ev: code niagara-help/versions/4.15.6.20/source/baja/javax/baja/util/Invocation.java:85-91]` `[ev: corpus B1316 §1316.1]`
- `BPermissions.all` fallback `[ev: code niagara-help/versions/4.15.6.20/source/baja/javax/baja/sys/BComponent.java:1979-1982]` `[ev: code …/naming/OrdTarget.java:465-467]` `[ev: corpus B1316 §1316.2]`
- Link path `[ev: code organized/baja/baja/vineflower/com/tridium/sys/schema/ComponentSlotMap.java:1683-1686]` vs `[ev: code types/logic.md:111-112]` `[ev: corpus B1316 §1316.3]`
- Command keys `[ev: code organized/docSource/docSource-doc/extracted/bajaui-wb/javax/baja/ui/Command.java:61-64]`; nav consumers `[ev: code …/baja/javax/baja/user/BUser.java:1030]` vs `[ev: code types/structure.md:264]`; BFormat `[ev: code …/baja/javax/baja/util/BFormat.java:62-72,372,443,666,744-750,755,1095-1106]` vs `[ev: code types/logic.md:406-414]` `[ev: corpus B1317]`
- kitControl `[ev: code niagara-help/versions/4.15.6.20/source/kitControl-rt/com/tridium/kitControl/util/BSwitch.java:127,317,336]` `[ev: code …/logic/BLogic.java:113]` `[ev: code …/logic/BHysteresis.java:35,292]` `[ev: corpus B1319]`; Expr null `[ev: corpus B1336 §1336.1 row 5]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | State the two coalescing winners: `post()` is FIRST-wins (post a no-arg "recalculate" and read current slots inside it); a `CoalesceQueue` of `Invocation`s is LAST-wins; queued ASYNC actions run with a null Context — never read `cx.getUser()` there. Confirms tjd Δ3. Lint: no. | `types/actions.md` § threading | `[ev: corpus B1316 §1316.1]` |
| Δ2 | Add: a null Context (or no user) yields `BPermissions.all`, so `OrdTarget.can*` all pass; servlets resolve with the `WebOp`, actions with the caller's Context, never `null` before a permission check. Lint: partial — WARN on `.resolve(<x>, null)` in a file that calls `.canRead(`/`.canWrite(`. | `types/security.md` + `types/logic-authoring.md:73` | `[ev: corpus B1316 §1316.2]` |
| Δ3 | Extend rac Δ10: `add(name, BLink)`, `linkTo()` and `makeLink()` never run `checkLink`; call `target.checkLink(source, sourceSlot, targetSlot, cx)` and require `isValid()` before adding a link in code. Lint: yes — WARN when a file adds a `BLink`/calls `linkTo(` without `checkLink(`. | `types/logic.md:111-143` | `[ev: corpus B1316 §1316.3]` |
| Δ4 | Add a Commands card: `new Command(owner, lex, keyBase)`/`ToggleCommand` read `keyBase.label`, `.icon`, `.accelerator`, `.description`; ship all four; a bad accelerator only logs. Lint: partial — check the four suffixes per key. | `types/wb-widgets.md` (new § Commands) | `[ev: corpus B1317 §1317.1]` |
| Δ5 | Extend nsto Δ5: delete "Workbench and the station discover `.nav` files … and merge them"; nav files load only via `BUser.navFile` (one file) or the profile's mobile nav file; assign the ORD on the user/prototype; files live under Files. Lint: yes — FAIL on the phrase "discover `.nav` files". | `types/structure.md:253-287` | `[ev: corpus B1317 §1317.2]` `[ev: corpus B1328 row 5]` |
| Δ6 | Extend rac Δ9 with the line-level facts: effective order slot → 6 reflect steps → `BIFormatPropertyHandler`; `time()`/`user()` exact, no argument (chain a format after `time()`); `substring(i)`, `(-n)`, `(a,b)` on the previous value; `?` = fallback script; no `ReflectionException`. Lint: yes — WARN on `%time([^)]` and `%user%`. | `types/logic.md:399-418` | `[ev: corpus B1317 §1317.3]` |
| Δ7 | Extend rac Δ5: kitControl status families — Select/MuxSwitch/Switch copy the selected value with `bits & propagateFlags` and HOLD `out` (+ invalid facet) on a bad selector; Hysteresis/Limiter/math compute with `bits & propagateFlags`; defaults (`nullStatus` / `ok`) mask FAULT/DOWN/STALE — set `fault,down,stale,disabled` downstream of probes. Lint: partial — bog check for default flags on blocks fed by proxy points. | `types/logic.md:252-256`, `:335` (LC3/LC4) | `[ev: corpus B1319 §1319.1]` |
| Δ8 | Note `BHysteresis`/`BLimiter` are 4.15+; swapped limits fault; the hysteresis latch resets to false on restart. Lint: yes — WARN when a <4.15 target references them. | `types/logic.md` (version note) | `[ev: corpus B1319 §1319.3 D3]` |
| Δ9 | Require an explicit NaN/null fallback before any staging comparison (vendor Expr converts null to NaN only for Double/Float inputs and keeps the last value for Integer/Long). Lint: partial — `Double.isNaN` guard grep near comparisons. | `types/logic-authoring.md` (staging) | `[ev: corpus B1336 §1336.1 row 5]` |

## Lessons
- A permission check is only as strong as the Context passed in; "null" means "everything" in baja.
- Two framework queues with the same word ("coalesce") keep opposite invocations — name the winner in the kit, never the verb.
- A block default that masks status bits turns a dead sensor into a valid reading; the kit must state each family's default, not one rule.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-15-devguide415-r2-runtime-contracts-formats.md | kit | 2026-10-15 | pending | 9 |`
