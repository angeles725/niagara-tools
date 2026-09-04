# Type: pure logic (rt control) — GROWING (control core proven on ColdRoomPan, 2026-08-31)

A control module: `BComponent`s with real control logic, no UI. Exemplar to READ before building: ColdRoomPan-rt (`BColdRoom`, `BEvaporatorUnit`, `BDefrostController`).

Seed pointers for the surface not yet proven; the sections below are proven from the ColdRoomPan-rt build:

- **Control engine:** logic runs on `execute()` / `changed()` / `atSteadyState()` on the engine thread — guard so it never throws there. Study the Control Engine in corpus (`corpus-nav show 6`) + kitControl (`BTstat` hysteresis: `sp ± diff/2`, HOLD between = deadband). `differential` is a band in degrees.
- **Links, not polling:** inputs arrive via Niagara links; read input `BStatus` and fail safe on invalid/null.
- **Slots:** computed outputs = `TRANSIENT|SUMMARY|READONLY`; config = `SUMMARY|OPERATOR`. Same facet/BDouble rules as METHODOLOGY.md.
- **No servlet, no `-ux`** unless paired with a dashboard (then also read types/dashboard.md).

## Safety fail-modes & timers
- **Anchor a free-running interval to a persistent clock, not to `atSteadyState`:** hold the last event time in a `HIDDEN` `BAbsTime` slot (`defaultValue = BAbsTime.NULL`, NOT transient); on re-arm compute `elapsed = Clock.millis() - getLastEventTime().getMillis()` and schedule `max(interval - elapsed, 0)` (`isNull()` → full interval). Rescheduling the full interval on every `atSteadyState` starves the timer across restarts. Clock/BAbsTime API [CERT] (`javax/baja/sys/Clock.java`, `BAbsTime.java`); the restart re-arm [INFER · pending station smoke-test] [ev: retro rt-hardening #2].
- **Anchoring is NOT enough — a self-armed timer MUST arm from `started()`, not only `atSteadyState()`:** `atSteadyState()` is a bootstrap-only callback (fires once, after the steady-state timeout); `started()` fires whenever the component's running state → true, INCLUDING a late mount onto an already-running station (commissioning). Arm in BOTH: `atSteadyState(){ if(isRunning()) arm(); }` + `started(){ super.started(); if(Sys.atSteadyState()) arm(); }` (+ optional `clockChanged(BRelTime){ arm(); }`). Verified canonical: the anti-pattern "override atSteadyState AND NOT started" has ZERO hits across the entire Tridium first-party corpus — Tridium NEVER arms a timer only in atSteadyState. A ColdRoomPan `BDefrostController` armed only in `atSteadyState` never desescharcha when mounted late; CompPan `BCompressorControl` freezes lead/lag rotation + start-prove. [CERT · BComponent.java:333-341/380-384, BTimeTrigger.java:213/221/294] [ev: corpus B729, retro 2026-09-03-self-firing-timer-needs-started-not-only-atsteadystate.md]
- **A sensor-fault control posture is an operator config slot, not a baked constant:** expose hold-last / force-cool / force-off as a `SUMMARY|OPERATOR` slot defaulting to the current behavior, so hardening never silently changes control on a live plant. [ev: retro rt-hardening #4]
- **Alarm-limit slots the control does not read are correct by design — alarms NOTIFY, they never STOP control:** do not wire `roomHighAlarmLimit` / `evapHighAlarmLimit` / … into the control decision; they belong to the facade/alarm source, not the control path. [ev: bitácora rt-hardening §4C]

## Staging & interlocks
- **An interlock that must not drop concurrent requests uses a FIFO queue, not a single slot:** `Deque<Integer>` (`ArrayDeque`, stdlib, no Baja type); dedupe active/pending/queued on request; `pollFirst` on terminate. A single `waitingUnit` int silently loses the 3rd request. [ev: bitácora rt-hardening §1]
- **Guard a capability-scoped resource token by that capability before granting it:** units without `hasDefrost` were holding the interlock token for its full duration and blocking others — filter `getHasDefrost()` before the token is granted. [ev: bitácora rt-hardening §4A]
- **A per-output HOA mode is a `TRANSIENT` double 0/1/2 (restart → Auto), and the priority is structural — defrost > HOA > auto:** re-apply the outputs whenever the mode changes. [ev: bitácora 5cuartos §5]

## Linking across custom modules
- **A value linked ACROSS two custom modules is a plain `double` (0/1/2), never a shared frozen-enum type:** an enum link requires the identical type on both ends, which forces module-B-rt to depend on module-A-rt, and the custom-module dependency DSL is non-trivial (`compileOnly(files(...))` does not reach the plugin classpath; the plugin auto-includes only Tridium modules). A `double` links with zero dependency; Workbench shows 0/1/2. The deleted `BHoaMode` was such an enum — a leftover reference surfaced live on the JACE as `Missing class ColdRoomPan:HoaMode`. [ev: bitácora 5cuartos §5]

## Logging
- **A plain non-`BObject` helper class compiles and bundles into the jar — use one for logging:** N4 has no `BLoggingService`; a shared `java.util.logging.Logger.getLogger("<module>")` lets an engine-thread handler log-and-swallow a throwable instead of discarding it. [ev: retro rt-hardening #3]

## Regenerating slots
- **Adding a slot to an already-generated component regenerates cleanly:** hand-write the annotation + a matching AUTO stub that only has to compile, then run `slotomatic` — it re-derives the AUTO region and updates the type hash. Slotomatic is authoritative; a fix only in the generated region reverts on the next regen. [ev: retro rt-hardening #1]

## Tridium rt idioms to adopt (full catalog: corpus B730)

Distilled from a docSource survey of control-rt/kitControl-rt (BControlPoint, BQuadMath, BRaiseLower, BSequence, …). Adopt these; the corpus block B730 has exemplar file:line for each.
- **Compute into a working value, commit only on real change** — mutate the passed `BStatusValue`, NEVER write `out` directly (re-enters `changed()` → loop). `if(!out.equivalent(working)) out.copyFrom(...)`.
- **Degrade honestly** — check `getStatus().isValid()` on every sensor; force output `null`/`fault` when inputs insufficient; propagate aggregated input status to the output (`out.setStatus(propagate(...))`). A bad probe → fault, not a false reading that trips a bogus alarm.
- **`changed()` = `super` + `if(!isRunning())return` + dispatch on WHICH slot + a deadband/significance guard before writing a slot back** (kills relay chatter through a feedback link).
- **Timer callbacks are `HIDDEN|ASYNC` actions; cancel-before-reschedule; cancel+null every ticket in `stopped()`.**
- **Flags**: `TRANSIENT` runtime state · `READONLY` computed outputs (pair both for live outputs) · `SUMMARY`/`OPERATOR` tunables · `DEFAULT_ON_CLONE` for calc state (so a cloned room/evaporator doesn't inherit stale numbers) · `ASYNC` timer actions · `FAN_IN` multi-link inputs.
- **`getSlotFacets` projection** of a `facets` config slot onto outputs (units/precision once) + **range facets** on inputs, still clamp defensively in code.
- **Pure-logic split for reusable formulas** (like `ColdRoomControl.decideCall`) — the only part that gets real unit tests; everyday logic stays inline.
- **GOTCHA**: `catch(Throwable)+log` is NOT automatic — the framework only wraps `BControlPoint.executeExtensions`. In your OWN `changed()`/timer handlers you MUST self-guard or one exception corrupts engine state. (Our modules already do.)

Verify with METHODOLOGY.md + build-verify.md. TODO: deepen the `execute()` / `changed()` cycle timing and multi-stage coordination from further builds.

See also: `docs/how-to-create-coldroom-module.md` (end-to-end ColdRoomPan build); corpus **B729** (timer lifecycle: started/atSteadyState/clockChanged) + **B730** (rt idioms catalog).
