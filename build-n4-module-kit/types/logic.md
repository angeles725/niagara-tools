# Type: pure logic (rt control) — GROWING (control core proven on ColdRoomPan, 2026-08-31)

A control module: `BComponent`s with real control logic, no UI. Exemplar to READ before building: ColdRoomPan-rt (`BColdRoom`, `BEvaporatorUnit`, `BDefrostController`).

Seed pointers for the surface not yet proven; the sections below are proven from the ColdRoomPan-rt build:

- **Control engine:** logic runs on `execute()` / `changed()` / `atSteadyState()` on the engine thread — guard so it never throws there. Study the Control Engine in corpus (`corpus-nav show 6`) + kitControl (`BTstat` hysteresis: `sp ± diff/2`, HOLD between = deadband). `differential` is a band in degrees.
- **Links, not polling:** inputs arrive via Niagara links; read input `BStatus` and fail safe on invalid/null.
- **Slots:** computed outputs = `TRANSIENT|SUMMARY|READONLY`; config = `SUMMARY|OPERATOR`. Same facet/BDouble rules as METHODOLOGY.md.
- **No servlet, no `-ux`** unless paired with a dashboard (then also read types/dashboard.md).

## Safety fail-modes & timers
- **Anchor a free-running interval to a persistent clock, not to `atSteadyState`:** hold the last event time in a `HIDDEN` `BAbsTime` slot (`defaultValue = BAbsTime.NULL`, NOT transient); on re-arm compute `elapsed = Clock.millis() - getLastEventTime().getMillis()` and schedule `max(interval - elapsed, 0)` (`isNull()` → full interval). Rescheduling the full interval on every `atSteadyState` starves the timer across restarts. Clock/BAbsTime API [CERT] (`javax/baja/sys/Clock.java`, `BAbsTime.java`); the restart re-arm [INFER · pending station smoke-test] [ev: retro rt-hardening #2].
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

Verify with METHODOLOGY.md + build-verify.md. TODO: deepen the `execute()` / `changed()` cycle timing and multi-stage coordination from further builds.

See also: `docs/how-to-create-coldroom-module.md` (end-to-end ColdRoomPan build).
