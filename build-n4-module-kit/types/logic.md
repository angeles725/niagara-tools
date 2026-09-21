# Type: pure logic (rt control) (control core proven on ColdRoomPan, 2026-08-31)

A control module: `BComponent`s with real control logic, no UI. Exemplar to READ before building: ColdRoomPan-rt (`BColdRoom`, `BEvaporatorUnit`, `BDefrostController`). Start-here reading for rt: **`corpus-index.md`** → B744 (what an RT block is) + B737 (composition) + B730/B729 (idioms/timers), all P0.

Seed pointers for the surface not yet proven; the sections below are proven from the ColdRoomPan-rt build:

- **Control engine:** logic runs on `execute()` / `changed()` / `atSteadyState()` on the engine thread — guard so it never throws there. Study the Control Engine in corpus (`corpus-nav show 6`) + kitControl (`BTstat` hysteresis: `sp ± diff/2`, HOLD between = deadband). `differential` is a band in degrees.
- **Engine-thread cost — HogsPage diagnostic:** visit `/spy/sysManagers/engineManager?hogs` (the `$HogsPage`) to see the top engine-thread consumers; the jstack thread to trace is `"Niagara Engine"`. Five-tier severity: `< 10 ms` OK · `10–50 ms` attention · `50–200 ms` move work to a driver worker · `200–1000 ms` critical · `> 1000 ms` probable blocking I/O. `$HogsPage` resets on station restart — collect while the station is live. `[ev: corpus B31 §31.1.5]`
- **Links, not polling:** inputs arrive via Niagara links; read input `BStatus` and fail safe on invalid/null. Empirical polling limits (N4 decompilation + B15; not Tridium-published): 1–2 k points @1 s safe; 5 k @5 s safe; 5 k @1 s marginal; above 5 k @1 s engine overload risk accumulates. `[ev: corpus B31 §31.2.1, B15]`
- **Slots:** computed outputs = `TRANSIENT|SUMMARY|READONLY`; config = `SUMMARY|OPERATOR`. Same facet/BDouble rules as METHODOLOGY.md.
- **No servlet, no `-ux`** unless paired with a dashboard (then also read types/dashboard.md).
- **A control `BComponent`'s station MOUNT/ORD is integrator-placed config, NOT derivable from module source:** a `BComponent` does not live at a fixed path by default — the path (`Programacion/CompresorControl`) is what the integrator chose at commissioning. Do not fabricate a plausible path; state it is unknown and get it from a live oBIX nav or the operator. [ev: retro dashboardpan-2d-to-3d-port Δ4]

## Safety fail-modes & timers
- **The 8 timer defense-in-depth layers (index — most already below; each stated once):** (1) anchor a free-running interval to a persistent `BAbsTime`, not `atSteadyState` [see "Anchor a free-running interval…"]; (2) arm in BOTH `started()`+`atSteadyState()` `[ev: corpus B729]`; (3) floor every delay `> 0` before `Clock.schedule`/`schedulePeriodically` [see "Guard any Clock.schedule…"] `[ev: corpus B801]`; (4) an INDEPENDENT liveness monitor on the producer's `lastTick` (stall > 3× period → fault + alarm) — the layer Tridium does NOT ship, author design `[ev: corpus B812]` (INFER); (5) ONE shared cancel path at both command edges + mode enter/exit, not only `stopped()` [see "Cancel EVERY actuation ticket…"]; (6) cancel every ticket in `stopped()` `[ev: corpus B775]`; (7) a manual run-now action + surfaced/safe-defaulted preconditions + a seeded first fire [see "A time-gated auto control needs THREE things…"]; (8) exemplar to copy — the `BAbstractAlarmMonitor`/`BTimeTrigger` self-heal `[ev: corpus B775 §775.6]`.
- folded as code: toolbelt/lint-delays.sh — layer 2-3 floor check is statically lintable; exit 1 on any ≤ 0 delay expression. `[ev: corpus B801]`
- **The delay floor MUST be INLINE in the `Clock.schedule` argument — a variable floor is NOT detected by `lint-delays`:** `lint-delays.sh` traces only an inline arithmetic expression in the schedule call argument. A floor computed in a preceding variable (`long delay = Math.max(ms, MIN); Clock.schedule(delay)`) passes the lint (the lint sees `delay`, a non-literal variable reference, and skips it) but is NOT protected. The only gate-safe pattern is the one-liner `Clock.schedule(Math.max(delayMs, MIN_DELAY_MS))`. T7 regression: a variable-floor refactor silently bypassed the lint and was caught only at runtime. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ6]`
- **GC-induced engine timer drift warning:** Niagara ships with ParallelGC (JDK 8 default); a Full GC stop-the-world pause (500 ms–2 s on a 1 GB heap) stops the engine thread and can skip a timer callback entirely — timer periods < 1 s are at risk. For safety-critical timing, add to `nre.properties`: `-XX:+UseG1GC -XX:MaxGCPauseMillis=200` (soft target — Full GC can still exceed it; G1GC reduces the risk but does not eliminate it). `[ev: corpus B31 §31.3.8]`
- **Anchor a free-running interval to a persistent clock, not to `atSteadyState`:** hold the last event time in a `HIDDEN` `BAbsTime` slot (`defaultValue = BAbsTime.NULL`, NOT transient); on re-arm compute `elapsed = Clock.millis() - getLastEventTime().getMillis()` and schedule `max(interval - elapsed, 0)` (`isNull()` → full interval). Rescheduling the full interval on every `atSteadyState` starves the timer across restarts. Clock/BAbsTime API [CERT] (`javax/baja/sys/Clock.java`, `BAbsTime.java`); the restart re-arm [INFER · pending station smoke-test] [ev: retro rt-hardening #2].
- **Anchoring is NOT enough — a self-armed timer MUST arm from `started()`, not only `atSteadyState()`:** `atSteadyState()` is a bootstrap-only callback (fires once, after the steady-state timeout); `started()` fires whenever the component's running state → true, INCLUDING a late mount onto an already-running station (commissioning). Arm in BOTH: `atSteadyState(){ if(isRunning()) arm(); }` + `started(){ super.started(); if(Sys.atSteadyState()) arm(); }` (+ optional `clockChanged(BRelTime){ arm(); }`). Verified canonical: the anti-pattern "override atSteadyState AND NOT started" has ZERO hits across the entire Tridium first-party corpus — Tridium NEVER arms a timer only in atSteadyState. A ColdRoomPan `BDefrostController` armed only in `atSteadyState` never desescharcha when mounted late; CompPan `BCompressorControl` freezes lead/lag rotation + start-prove. [CERT · BComponent.java:333-341/380-384, BTimeTrigger.java:213/221/294] [ev: corpus B729, retro 2026-09-03-self-firing-timer-needs-started-not-only-atsteadystate.md]
- **A sensor-fault control posture is an operator config slot, not a baked constant:** expose hold-last / force-cool / force-off as a `SUMMARY|OPERATOR` slot defaulting to the current behavior, so hardening never silently changes control on a live plant. [ev: retro rt-hardening #4] **Carve-out:** "default to current behavior" applies to a change on an ALREADY-COMMISSIONED path only; a brand-NEW safety/startup slot has no operator setting to protect and must default to the SAFE posture (sentinel `0 = AUTO`) — safety wins, and you flag the behavior change in the retro + version note. [ev: retro soft-start · L15]
- **Alarm-limit slots the control does not read are correct by design — alarms NOTIFY, they never STOP control:** do not wire `roomHighAlarmLimit` / `evapHighAlarmLimit` / … into the control decision; they belong to the facade/alarm source, not the control path. [ev: bitácora rt-hardening §4C]
- → **When an ML model proposes a setpoint or stage command, the same doctrine applies:** wrap the proposal in the PROPOSE-VALIDATE gate (range clamp, rate-of-change limit, anti-short-cycle, HOA-OFF lockout, fallback-on-NaN) before it reaches the control core. The model is an input to the control system, never a bypass of its protection layers. See `types/ml-libraries.md` for the full safety architecture and the library selection map.

- **A `Long.MIN_VALUE` "never happened" time sentinel MUST be guarded at EVERY subtraction site:** `now - Long.MIN_VALUE` overflows negative for any real `Clock.millis()`, so a `(now - sentinel) >= delay` gate is false forever and the rack NEVER stages a compressor — silent, catastrophic, and it compiles and passes the verify gate (caught only by a pre-compile reviewer). Guard `sentinel == Long.MIN_VALUE || (now - sentinel) >= delay`, or use `0` / an explicit DISABLED state instead of the extreme sentinel. [ev: retro comppan-fase1 · L1]
- **Cross-refs (folded into `METHODOLOGY.md`):** never retype a live slot — bog boot crash (§Schema/upgrade safety · S1); `0` on a protection limit means DISABLED not worst-case (§Domain correctness · L2).

- **Distinguish 'invalid reading' from 'bad value' — on a dead sensor assume the COMMANDED state, fault only on a VALID out-of-range reading:** `running[k] = ampsValid ? amps > thr : cmd[k]` — when you cannot measure, assume commanded; conclude "failed to start" (and latch a healthy unit out) ONLY on a valid low reading, or a dead amp sensor false-faults good equipment. [ev: retro comppan-fase1 · L3]
- **Latch a start/safety fault and require an operator `faultReset` slot; never auto-retry a failed rotating-equipment start:** auto-retrying a compressor that failed to start can damage the motor — expose a momentary `faultReset` (`SUMMARY|OPERATOR`, self-resetting); the unit stays out with alarm until the operator acknowledges. [ev: retro comppan-fase1 · L4]
- **Persist only state that must survive a restart; add `resetTransient()` called in `stopped()` so a disable→enable cold-starts:** persist run-hours (non-transient, seeded once at `atSteadyState` via a `hoursSeeded` flag) and keep commands/running/faults/timers transient — a disable→enable reuses the SAME instance (not a restart), so without the reset it resumes stale commands. [ev: retro comppan-fase1 · L7]
- **Arm the self-firing heartbeat tick in its OWN try/catch, BEFORE the first `execute()`:** if `armTick()` runs after `execute()` in one try and that first `execute()` throws, the tick is skipped and the control degrades to reactive-only forever. [ev: retro comppan-fase1 · L8]
- **Cancel EVERY actuation ticket on ONE shared cancel path, invoked at BOTH command edges and at mode enter/exit — not only in `stopped()`:** once a unit grows from one ticket to three (start / stop-run-on / defrost-fan-off), a stale run-on fire otherwise turns the fan off mid-cooling. [ev: retro coldroompan-fan-mode-defrost · L10]
- **Ship the whole mechanism with an unvalidated safety-critical data function STUBBED to NaN (provably inert), never invented placeholder numbers:** when a feature needs field data not yet validated (e.g. an R404A P-T curve), a NaN stub keeps the adapter on its fixed fallback so zero invented numbers reach the plant; activation later is a one-function fill-in. [ev: retro comppan-fase3 · L13]

- **A time shown in a POLLED UI is a READONLY `BAbsTime` ANCHOR set at the event edge, never a stored `BRelTime` "remaining":** a `BRelTime` freezes between engine executions; store the event instant as a `TRANSIENT|SUMMARY|READONLY` `BAbsTime` and let the reader/SPA compute `now − anchor` each poll (the reader needs an AbsTime accessor — see `types/dashboard.md`). [ev: retro process-timers · L16]
- **Reserve `Flags.HIDDEN` for engine timer callbacks; anything an operator / oBIX / HMI must invoke needs a NON-HIDDEN action:** a `@NiagaraAction(flags=HIDDEN)` shows an empty Workbench Actions menu and 0 `<op>` over oBIX — expose a public `forceDefrost` / `runNow` that calls the same routine. [ev: retro hidden-actions · L17]
- **Guard any `Clock.schedule` reachable via a linked input with `if(!Sys.atSteadyState()) return;`, set the anchor/state slot BEFORE scheduling, and validate `period > 0` before `schedulePeriodically`:** a linked input propagating during `activateLinks` → `changed()` → `execute()` → `Clock.schedule` throws `NotRunningException` (the `isRunning()` guard does NOT cover it), and a throw after a null anchor leaves it null AND unarmed. [ev: retro self-firing-timer · L18]
- **A time-gated auto control needs THREE things or it silently "never enters": a manual run-now action, its enabling preconditions surfaced (or safe-defaulted), and an anchored/seeded first fire so a restart doesn't reset the countdown:** the defrost "never entered" was all three (`hasDefrost` default-false with no field, interval re-armed full on every restart with a null `lastDefrostTime`, no manual trigger) — design gaps, not sequence bugs. [ev: retro process-timers · L20]

## Schema-safe evolution
- **When a slot's shape MUST change and ADD-new-and-link is insufficient, use a `started()` self-migration:** (a) ADD the new typed slot; (b) in `started()` (or `atSteadyState()` if the station may carry saved data to migrate), detect the orphaned dynamic slot by name — `get(oldName) != null && getSlot(oldName).isDynamic()` — copy its value to the new slot, then remove the old slot; (c) leave the old slot deprecated-but-present for one release cycle until confirmed no station carries the old form. `BModule` is `final` with no `upgrade()` or `migrate()` callback; whole-station migration is the offline `migration-rt` tool (`BIFileMigrator` + converter SPIs) and is NOT available in-component. [ev: corpus B754 §754.3 · started-self-migration-recipe]

## Write-path & overlap `[ev: corpus B816]`
- **`set()` runs on the CALLING thread; only the raw value store is locked:** a servlet/Workbench write and an engine `execute()`/Clock callback on the SAME component serialize ONLY for the raw store (`synchronized(this.instance)`, last-writer-wins, no torn value). Their CALLBACKS — `changed()` and link propagation — run OUTSIDE that lock, synchronously, and CAN interleave; `changed()` is re-entrant (a `changed()` that calls `set()`/`schedule()` recurses on the stack). Overlap bugs are real and live in the CALLBACKS, not the store. `[ev: corpus B816]`
- **A dashboard write to a LINK-TARGET slot LANDS, then is silently overwritten:** the `Flags.LINK_TARGET` bit is advisory metadata checked ONLY at link-CREATION in the wiresheet — never by `set()` nor `canWrite()`. A manual write to a link-driven slot sticks only until the next source propagation re-`set`s it (no rejection, no exception). A write to a NON-linked `OPERATOR` slot does stick. The UI must NOT imply a link-target write persisted. `[ev: corpus B816]`
- **A `Transaction` is NOT cross-thread atomic:** it queues ops and replays them FIFO through the same per-slot lock; there is no global lock across the batch and no cross-batch isolation — the engine thread can observe a half-written multi-slot state. Do not rely on a Transaction to hide interleaving. `[ev: corpus B816]`

## Staging & interlocks
- **An interlock that must not drop concurrent requests uses a FIFO queue, not a single slot:** `Deque<Integer>` (`ArrayDeque`, stdlib, no Baja type); dedupe active/pending/queued on request; `pollFirst` on terminate. A single `waitingUnit` int silently loses the 3rd request. [ev: bitácora rt-hardening §1]
- **Guard a capability-scoped resource token by that capability before granting it:** units without `hasDefrost` were holding the interlock token for its full duration and blocking others — filter `getHasDefrost()` before the token is granted. [ev: bitácora rt-hardening §4A]
- **A per-output HOA mode is a `TRANSIENT` double 0/1/2 (restart → Auto), and the priority is structural — defrost > HOA > auto:** re-apply the outputs whenever the mode changes. [ev: bitácora 5cuartos §5]

- **With amps + command you get TWO near-free alarms — add BOTH:** CMD ON + no amps past the prove-delay = failed start; CMD OFF + amps present = stuck contactor (critical). Easy to write the first and forget the second. [ev: retro comppan-fase1 · L5]
- **A mode that holds an output OUTSIDE the command edges needs its own apply method wired into every setter:** a `continuous` fan runs with no edge firing, so `applyFanRunMode()` must be called from `atSteadyState` + `changed(modeSlot)` + defrost-exit, and every other writer of that output (e.g. `applyHoaOutputs`) must honor it — audit every code path that sets a decoupled output. [ev: retro coldroompan-fan-mode-defrost · L11]
- **Ship an additive control phase DORMANT behind a config default (e.g. `suctionBand > 0`) so a fresh jar behaves exactly like the prior phase until commissioned:** the prior test suite leaves that config at default, stays green unchanged, and guards backward-compat for free. [ev: retro comppan-fase2 · L12]
- **For position-derived behavior (staggered startup), self-order by a GUARDED sibling tree-walk, not a coordinator component:** walk `getParent()`/`getChildren(BColdRoom.class)` in slot order inside a `try/catch → return 0` to derive plant-wide position — no new `@NiagaraType` coordinator + palette entry. [ev: retro soft-start · L14]

- **HOA manual-override pattern for a control component:** one `double` `@NiagaraProperty` per output (`SUMMARY|OPERATOR`, `0=auto / 1=on / 2=off`), passed into the pure core via an overload with an all-AUTO array (keeps prior tests unchanged); apply the override LAST — OFF excludes from rotation, ON forces true ONLY if `(now − cmdSince) >= minOff` (never short-cycle); add the mode slots to `changed()`. [ev: retro hoa-manual-override · L22]

## Zero-demand / idle state `[ev: corpus B819 §819.4–819.5]`

Six-point doctrine for any staged or modulated control component. Author the idle/demand gate before writing the staging or modulation logic.

1. **Demand is a first-class gate — never skip it on a valid setpoint.** If the process variable is in-band and demand is zero, the actuator is off. Do not conflate "a setpoint is present" with "there is demand."
2. **NaN / invalid setpoint ≠ demand — guard with `Double.isFinite` or `getStatus().isValid()`.** See anti-pattern below.
3. **Every staged process declares an explicit idle state.** The idle state is a named phase in the phase machine (see §RT control logic §805.9 STATES layer), not a side-effect of demand being zero. Calling it out prevents control-path fallthrough.
4. **Expose a "why running" surface.** A `TRANSIENT|SUMMARY|READONLY` `String` slot (e.g. `activeReason`) updated on every demand transition lets an operator know whether the rack is running because of demand, manual override, or pre-cooling. A `null` / empty string on idle.
5. **HOA OFF lockout dominates — respect it even when demand is non-zero.** Off lockout overrides demand; a demand that would stage up while OFF is held must NOT fire.  The dominant order is: OFF lockout > demand gate > staging logic.
6. **minOn / stageDelay guards apply in BOTH staging directions.** A minOn floor prevents short-cycling on stage-up; an equivalent stage-down timer prevents hunting. Author them together — a stage-up guard with no stage-down guard ships half the protection.

**The NaN setpoint hazard (B819 §819.3):** an unguarded numeric setpoint (`suctionSetpoint`, `tempSetpoint`) causes a silent modulation freeze. The demand gate turns the rack off, but the PID / band logic silently continues emitting control signals because demand was never NaN-gated. Always guard before entering the staging / modulation path:

```java
if (!Double.isFinite(setpoint) || !setpointSlot.getStatus().isValid()) {
    // → idle: turn off, clear demand, return
}
```

`getStatus().isValid()` covers `BStatus.NULL` / `BStatus.FAULT` / `BStatus.STALE`; `Double.isFinite` covers NaN and ±Infinity from a detached link. Both guards are needed — a linked slot can carry a finite value with an INVALID status when the source is faulted.

## Linking across custom modules

### Link lifecycle — gating and reacting `[ev: corpus B958 §958.1–958.2]`

- **`doCheckLink` gates incoming links before creation:** override
  `doCheckLink(BComponent source, Slot sourceSlot, Slot targetSlot, Context cx)` in your component to
  validate whether a proposed link is legal.  Return `LinkCheck.makeInvalid("reason")` to block it;
  `LinkCheck.makeValid()` to allow it.  The framework calls this before any link is created in the
  wiresheet or programmatically.

  ```java
  @Override
  protected LinkCheck doCheckLink(BComponent source, Slot sourceSlot,
                                  Slot targetSlot, Context cx) {
      if (targetSlot == myInputSlot) {
          if (!source.getType().is(BRequiredType.TYPE))
              return LinkCheck.makeInvalid("source must be a BRequiredType");
      }
      return LinkCheck.makeValid();
  }
  ```

- **`added(Property, Context)` / `removed(Property, BValue, Context)` react to link creation/removal:**
  these callbacks fire whenever a link is added to or removed from the component.  Guard with
  `bValue instanceof BLink && isRunning()` so you only react to link additions during a running session
  (not slot loads at startup). Use `removed(Property, BValue, Context)` symmetrically to tear down
  anything `added()` set up.

- **INDIRECT-link rule — a programmatically added reciprocal link MUST use the `true` (indirect) flag:**
  when `added()` creates a back-link (e.g. wiring the source's offset back into this component), use:

  ```java
  BOrd sessionOrd = sourceComponent.getOrdInSession();
  add(null, new BLink(sessionOrd, sourceSlot.getName(), mySlot.getName(), true));
  //                                                                       ^^^^
  //                                         true = INDIRECT (ORD-resolved)
  ```

  **Why indirect?** A direct link embeds a hard reference — if the target component starts before the
  source during a station restart, the link is unresolvable and silently breaks.  An indirect link holds
  the source as an ORD and resolves it lazily, surviving any startup ordering.  The first-party comment
  in `componentLinks-rt` states this verbatim: "we cannot guarantee the order in which the components
  will start when the station is restarted." `[ev: corpus B958 §958.2]`

- **A value linked ACROSS two custom modules is a plain `double` (0/1/2), never a shared frozen-enum type:** an enum link requires the identical type on both ends, which forces module-B-rt to depend on module-A-rt, and the custom-module dependency DSL is non-trivial (`compileOnly(files(...))` does not reach the plugin classpath; the plugin auto-includes only Tridium modules). A `double` links with zero dependency; Workbench shows 0/1/2. The deleted `BHoaMode` was such an enum — a leftover reference surfaced live on the JACE as `Missing class ColdRoomPan:HoaMode`. [ev: bitácora 5cuartos §5]
- **A discrete selector used ONLY as an internal slot is a `BFrozenEnum` (registered in `module-include.xml`), NOT a double — the plain-double rule above is for values linked ACROSS custom modules only:** e.g. `BFanMode` internal config is a frozen enum. [ev: retro coldroompan-fan-mode-defrost · L19]
- **`BIUnlinkableSlotsContainer`: implement to keep specific child slots operator-writable but never a link source or target** — `HIDDEN` removes a slot from ALL UI; `BIUnlinkableSlotsContainer` keeps it visible in the property sheet and writable by an operator but excluded from the wiresheet link picker. Use case: a setpoint or HOA override slot that an operator must set by hand and must never be driven by a link. `[ev: corpus B735 §735.4]`

### BConverter and cross-type link STRIP pattern `[ev: corpus B871, B928]`

- **One converter per link — no framework multi-hop:** `BConversionLink` holds exactly ONE `BConverter` slot. There is no framework facility for chaining converters across a single link. The `@Adapter(from, to)` registry auto-selects a converter by type pair.
- **Three status patterns:** PROPAGATE (`Status → Status` copies the `BStatus` bitmask end-to-end), STRIP (`Status → plain`: converter unwraps the value; status bitmask is NOT forwarded), INJECT (`plain → Status`: wraps a raw value in a `BStatus`).
- **STRIP gotcha — null/fault source holds last-good target, does NOT surface fault:** a `Status → plain` converter (e.g. `BStatusNumeric → double`) on a heterogeneous link fires inside `propagate`. When the source is `null` or faulted, the STRIP converter returns the **unchanged target value** (last-known-good) — NOT null and NOT NaN. The fault is silently absorbed at the conversion seam; the target slot never sees an invalid status. This is architecturally correct (preserves last-known-good across a type seam) but violates the "degrade honestly" rule if the author expects fault to propagate. **Rule:** always check `BStatus.isValid()` on the SOURCE side of a cross-type link before trusting the target value; do not assume a faulted source drives the target to null/fault. `[ev: corpus B871 §871.4]`
- **Enum → double bridge is ordinal-only (lossy):** an enum→double converter returns the enum's ordinal. The semantic tag is lost; a round-trip through a `double` destroys enum identity. `[ev: corpus B871 §871.2-3, B928]`
- **`BConversionLink` stale-converter rule — delete and re-create the link after retyping a source slot:** the converter is persisted at link-creation and chosen ONCE by the `@Adapter(from, to)` registry; when the source slot's type later changes, `propagate()` applies the OLD converter with no try/catch — the result is either a SEVERE-abort (throwing converter) or a silently wrong value (coercing converter). The framework does NOT re-select the converter on retype. Fix: after retyping the source, delete the link and create a new one so the registry picks the correct converter. `[ev: retro module-hardening-failure-modes-deltas Δ8]`

### Dangling link silent degradation `[ev: retro module-hardening-failure-modes-deltas Δ6]`

An unresolvable link (source ORD throws `UnresolvedException`) leaves the target slot at its stale value indefinitely. **No fault, no alarm, and no visible error are raised on the target side.** A cross-station link re-resolves only if the source component mounts at exactly the same ORD as was stored.

**Rule:** if a cross-station or cross-module value must be known-fresh, add an explicit staleness watchdog alongside the target slot — a `BNumericPoint` last-ok timestamp (or equivalent heartbeat) that fails-stale when the source stops updating. Audit inbound link ORDs with `bog-nav --slot` before renaming or moving a source component.

**Lint candidate:** `dangling-link-watchdog` — flag a cross-station link target with no nearby stale/heartbeat check in the same component.

## Composition & organization
- **Above ~12–15 flat slots, compose into child components (one per concern) and keep config separate from live-state — don't sprawl a flat slot wall:** Honeywell modules distribute by containment (one child BComponent per concern: timing / outputs / hoa / freeze), tunables in a frozen `config` child, value + `BStatus` on the component; a 25-slot flat `BEvaporatorUnit` is the smell. [ev: corpus B737/B749/B750 · L21]

### UX/legibility ranked checklist `[ev: corpus B748 §748.2]`

Six changes that fix wire-sheet and property-sheet overload, ordered by impact ÷ cost:

| # | Change | Cost | Notes |
|---|--------|------|-------|
| 1 | **SUMMARY pin curation** — real I/O = `SUMMARY`, internal state = non-summary, engine callbacks = `HIDDEN` | trivial · flag-only | Declutters the wire sheet immediately; no structural change. Do items 1 + 3 in the same pass. |
| 2 | **Compose flat slots into child components** (one child per concern) | medium · structural | One careful pass per module; the structural change that reduces the flat-slot wall permanently. |
| 3 | **Units/precision facets on every temp / pressure / percent slot** | low · facet-only | Applies to all SUMMARY-flagged numeric pins; pairs with item 1. |
| 4 | **Distinct icon per block type** | low · one SVG resource | See `types/logic-authoring.md §Adding a block icon`. Polish — do after items 1–3 are done. |
| 5 | **Pre-wired palette assembly templates** | low · resource-only | A `.ntpl` zip covering the most common wiring; see §Templates. Discoverability improvement. |
| 6 | **Semantic tag dictionary** | medium | See §Ship a tag dictionary above; enables NEQL queries and navigation. |

**Sequencing rule:** 1 + 3 are same-day flag/facet passes; 5 is a resource add; 2 is the structural one (plan carefully); 4 + 6 are polish and discoverability improvements.

## Logging
- **A plain non-`BObject` helper class compiles and bundles into the jar — use one for logging:** N4 has no `BLoggingService`; a shared `java.util.logging.Logger.getLogger("<module>")` lets an engine-thread handler log-and-swallow a throwable instead of discarding it. [ev: retro rt-hardening #3]

## Regenerating slots
- **Adding a slot to an already-generated component regenerates cleanly:** hand-write the annotation + a matching AUTO stub that only has to compile, then run `slotomatic` — it re-derives the AUTO region and updates the type hash. Slotomatic is authoritative; a fix only in the generated region reverts on the next regen. [ev: retro rt-hardening #1]
- **In the `build.sh`/slotomatic flow, annotation-ONLY is sufficient — no AUTO stub required:** `build.sh` runs slotomatic BEFORE javac, so a facade's 26 new `@NiagaraProperty` annotations compile with zero hand-written stubs. The AUTO stub is needed only when compiling WITHOUT slotomatic (IDE partial builds). Keep "annotation + stub" as belt-and-suspenders, not the requirement. [ev: retro freeze-stat-leds Δ2]
- **Slotomatic verbatim-expression triage — a javac error on the AUTO region means look at the ANNOTATION:** `@NiagaraProperty(defaultValue=…, flags=…)` values are emitted verbatim into the AUTO region as Java expressions without evaluation. A bad-type `defaultValue` (e.g. a `String` literal for a `double` property) or a mistyped `flags` constant (e.g. a typo in `Flags.*`) surfaces as a javac error on the GENERATED file. The root cause is always the annotation; do NOT edit the generated code — fix the `@NiagaraProperty` annotation and the error disappears on the next slotomatic pass. `[ev: retro module-hardening-failure-modes-deltas Δ13]`

## Pure-class extraction — test the decision before you wire it
- **Extract any inline timing/safety/decision logic in a BComponent into a ZERO-Baja pure class FIRST, test it (with bite assertions), confirm it compiles, THEN wire it into the BComponent:** the untested inline subsystem is exactly what ships the field bug — `BDefrostController`'s interval/interlock/terminate math was inline with no pure class and shipped the `started()`/interval defrost bug; a `nextDelayMs()` unit test (with a `|elapsed| > 3·interval → interval` future-clock guard) would have caught it. [ev: retro qa-stack · T2]
- **For a stateful controller, keep the decision core a stateful Baja-free instance with one `step(now, inputs, Cfg)`; the BComponent is a thin slot adapter** (reads slots → builds inputs → calls `step` → writes outputs). JUnit drives `step` over advancing time. Run it with `toolbelt/run-pure-test.sh <rt-dir> <pkg> <PureClass> <TestClass>`. [ev: retro comppan-fase1 · L6]
- **A control phase that changes only an INPUT to the existing decision needs NO change to the stable core — push the new logic to the BOUNDARY (the adapter that feeds the core):** e.g. floating suction recomputes the setpoint the core already consumes, done in the slot adapter, so the proven core control and its unit tests stay untouched. (Builds on the thin-adapter/stateful-core split above · L6, and the dormant-phase-behind-a-default rule · L12.) [ev: retro comppan-fase3-floating-suction]

## Tridium rt idioms to adopt (full catalog: corpus B730)

Distilled from a docSource survey of control-rt/kitControl-rt (BControlPoint, BQuadMath, BRaiseLower, BSequence, …). Adopt these; the corpus block B730 has exemplar file:line for each.
- **Compute into a working value, commit only on real change** — mutate the passed `BStatusValue`, NEVER write `out` directly (re-enters `changed()` → loop). `if(!out.equivalent(working)) out.copyFrom(...)`.
- **No-reentrancy invariant — outputs must NOT appear in the `changed()` filter:** the write→changed→execute loop cannot form iff `execute()` writes only outputs and `changed()` dispatches on inputs+config only (the structural version of the BRoomPanel flood bug, as an rt invariant). [ev: retro comppan-fase1 · L9]
- **Degrade honestly** — check `getStatus().isValid()` on every sensor; force output `null`/`fault` when inputs insufficient; propagate aggregated input status to the output (`out.setStatus(propagate(...))`). A bad probe → fault, not a false reading that trips a bogus alarm.
- **`changed()` = `super` + `if(!isRunning())return` + dispatch on WHICH slot + a deadband/significance guard before writing a slot back** (kills relay chatter through a feedback link). **`changed()` dispatch is SYNCHRONOUS on the same stack — a slot write from inside `changed()` immediately calls `changed()` again on the same call stack; an unconditional write produces a `StackOverflowError` (NOT a queue flood); a deadband guard is therefore a CORRECTNESS requirement, not just an efficiency hint; never write a slot unconditionally from `changed()`; `fireDirectCallbacks()==false` prevents this on an offline .bog.** `[ev: retro module-hardening-failure-modes-deltas Δ1]`
- **Timer callbacks are `HIDDEN|ASYNC` actions; cancel-before-reschedule; cancel+null every ticket in `stopped()`.**
- **Flags**: `TRANSIENT` runtime state · `READONLY` computed outputs (pair both for live outputs) · `SUMMARY`/`OPERATOR` tunables · `DEFAULT_ON_CLONE` for calc state (so a cloned room/evaporator doesn't inherit stale numbers) · `REMOVE_ON_CLONE` for dynamic result slots (discarded by `newCopy()` on clone/export/import — use for runtime-populated search results, learn entries, and transient state bags that must NOT replicate to a copy) `[ev: corpus B4 §4.3.3]` · `ASYNC` timer actions · `FAN_IN` multi-link inputs.
  - **TRANSIENT trap:** never combine `TRANSIENT` with `OPERATOR` on the same slot — `TRANSIENT` means not persisted to the `.bog`; `OPERATOR` means operator-writable config. The operator sets a setpoint, the station restarts, the setpoint silently reverts to default. `verify-module.sh --src` WARN: `transient-operator`. `[ev: corpus B755 §755.5, B4 §4.1.2]`
- **`getSlotFacets` projection** of a `facets` config slot onto outputs (units/precision once) + **range facets** on inputs, still clamp defensively in code.
- **Pure-logic split for reusable formulas** (like `ColdRoomControl.decideCall`) — the only part that gets real unit tests; everyday logic stays inline.
- **GOTCHA**: `catch(Throwable)+log` is NOT automatic — the framework only wraps `BControlPoint.executeExtensions`. In your OWN `changed()`/timer handlers you MUST self-guard or one exception corrupts engine state. (Our modules already do.)
- **Wrap every callback body and re-arm timers in `finally`:** `changed()`, timer callbacks, and lifecycle overrides are BARE — a thrown exception aborts the invocation and silently drops any `Clock.schedule` re-arm that follows. Pattern: `try { /* logic */ } catch (Throwable t) { logError("tick", t); } finally { rearmTick(); }` — the `finally` block guarantees the re-arm even on exception. A self-rescheduling timer that throws before re-arm silently stops for the rest of the session with no log entry, causing the control to degrade to reactive-only. `[ev: retro module-hardening-reference-cards-deltas Δ8]`

### Wire-sheet live-view recipe `[ev: corpus B747 §747.2]`

Three authoring choices determine the wire-sheet rendering and together produce a live, color-coded flow debugger at no extra cost:

1. **`SUMMARY` flag on a slot → visible pin row** — `SlotBarGlyph` gates pin-row visibility on `Flags.isSummary()` (SUMMARY=8, `Flags.java:12`); a slot without SUMMARY is invisible on the wire sheet regardless of type. Curate SUMMARY on real I/O slots; do not set it on every internal state slot.
2. **Units + precision facets on that slot → formatted live value in the pin row** — `PropertyBarGlyph.updateValueString()` reads the slot's facets and renders the live value in the pin row with the configured units and decimal precision. Add units/precision facets to every numeric slot that appears as a pin (the `getSlotFacets` projection, covered in the Flags bullet above).
3. **`BStatus` on the slot's value → pin row tinted by status color** — the same `PropertyBarGlyph` tints the row by the status color (fault=red, stale=yellow, override=magenta) via `getShowStatusColors()`. Refresh is push-on-change (`WsController.handleComponentEvent → glyph.changed(slot)`, `WsController.java:544-574`) — no poll.

**Practical rule:** curate SUMMARY on every real I/O slot, add units facets to every numeric pin slot, set `BStatus` on every output. All three are already best-practice for independent reasons; together they make the wire sheet a live diagnostic surface for free.

Verify with METHODOLOGY.md + build-verify.md. TODO: deepen the `execute()` / `changed()` cycle timing and multi-stage coordination from further builds.

## Slots — BStatus producers `[ev: corpus B736 §736.2–736.4]`

**8-bit BStatus reference** (BStatus.java:46–53; ok = 0, no bits set):

| Flag | Value |
|------|-------|
| `DISABLED` | 0x01 |
| `FAULT` | 0x02 |
| `DOWN` | 0x04 |
| `STALE` | 0x08 |
| `OVERRIDDEN` | 0x10 |
| `NULL` | 0x20 |
| `UNACKED_ALARM` | 0x80 |

Typed factory family (BStatus.java:51–140) — never construct a raw `BStatus(int)`:
`makeFault(s)`, `makeDown(s)`, `makeNull(s, addNull)`, `makeStale(s)`, `makeOverridden(s, addOverridden)`, `makeDisabled(s, state)`.

- **Set `makeOverridden(s, true)` on an output when it is forced by a HOA HAND or writable override** so the HMI and any subscriber can distinguish a forced value from an auto-computed one. The existing kit documents reading `OVERRIDDEN`; this is the producer side — the component applying the override must set the bit. `[ev: corpus B736 §736.4]`
- **Use `BStatus.makeNull(s, true)` when a sensor is absent or the computation is indeterminate** — write a NULL-status output, not a `0.0` with ok-status. A `0.0/ok` lie is a valid reading downstream; it can drive a false control decision (e.g. a 0 °C reading from a disconnected probe triggers unnecessary cooling). `[ev: corpus B736 §736.4]`
- **Gate control math on `isValid()`, not `isOk()`:** `isOk()` returns `false` whenever ANY status bit is set — including OVERRIDDEN and UNACKED_ALARM — so it drops control output even when the sensor value is numerically usable. `isValid()` returns `true` when bits = 0 OR when only override/unacked-alarm bits are set: the value is trustworthy for a control decision. Rule: use `isOk()` only for the strictest "no bits at all" check (e.g. a safety-latch gate); use `isValid()` as the control gate for all sensor reads and setpoint checks. Confusing the two silently drops control output on every manual HOA override — the fail-to-danger pattern B730/B650/B655 warns about. `[ev: retro module-hardening-reference-cards-deltas Δ2]`

## Propagation — `propagateFlags` on custom components `[ev: corpus B738 §738.3]`

**Add a `propagateFlags BStatus` slot (`SUMMARY|OPERATOR`, default `BStatus.ok`) to custom control components** to let operators tune which status bits (fault/stale/down/null) flow through to outputs — mirrors the `BKitNumeric` pattern (BKitNumeric.java:35). Without it the component hardcodes propagation unconditionally.

Recipe: `out.setStatus(getPropagateFlags().and(aggregatedInputStatus))` — AND the aggregated input status with the mask before setting the output. A `BStatus.ok` mask blocks all propagation; a mask with `FAULT` lets only fault bits through. The `SUMMARY|OPERATOR` flag exposes it in the property sheet; operators can tune per-site without a code change.

## RT control logic `[ev: corpus B805]`

- **§805.9 flowchart template** — design every rt component's internal logic as SEVEN layers: STATES (the phase machine: idle / running / fault / …), INPUTS (read all sensor slots, check `isValid()`, fail-safe on invalid), TIMERS (which `Clock.Ticket`s are active and their cancel points), CONTROL (the decision core — PID/deadband/staging), PROTECTIONS (limits that RAISE a latch, independent of control), OUTPUTS (what the component writes to actuator slots plus the fail-safe value on fault), FEEDBACK (status/health/alarm surface visible to the operator). Draw this template before writing any `execute()`. `[ev: corpus B805]`
- **§805.10 PID / loop anti-windup** — clamp `errorSum` to `[minOut, maxOut]` after every integration step; a saturated accumulator causes overshoot on setpoint change and slow recovery. On a NaN or invalid input: fault the output, hold the last valid command (bumpless transfer: on re-enable pre-load the integrator to the last output so the first step after recovery does not spike). `[ev: corpus B805]`
- **Deadband latch-on-cross** — latch ON when the error exceeds the upper deadband threshold; latch OFF only when the error drops below the lower threshold. Never compare the raw error against a single threshold — that causes relay chatter (e.g. a `BTstat` compares `sp + diff/2` vs `sp − diff/2`). `[ev: corpus B805]`
- **`execute()` / `changed()` split (no-reentrancy invariant)** — `execute()` runs the algorithm every scan: reads input slots → computes into a local variable → writes output slots ONLY if the result changed. `changed()` dispatches on WHICH input or config slot changed; outputs MUST NOT appear in the `changed()` slot filter or the write→changed→execute loop forms. `[ev: corpus B805]`
- **`BLatch` is an edge D-latch only** — it latches a boolean HIGH on the rising edge and does NOT implement an SR latch or a protection-level interlock. A protection latch (e.g. "stay locked out until operator reset") is AUTHOR-BUILT: a `boolean` field + a `faultReset` OPERATOR action that clears it. `[ev: corpus B805]`
- **`[CERT-negative]` No ODE / matrix / state-space facility in stock Niagara** — Tridium ships no physics-step or differential-equation integrator; a `step()` model for physical systems is beyond stock. Implement as a pure-class finite-state machine and test it off-station. `[ev: corpus B805]`
- **Health / feedback surface (R15.2)** — every rt component MUST expose: (1) a `status` slot (`SUMMARY|READONLY`) reflecting the current phase (OK / FAULT / STALL), (2) a `lastError` string slot updated when a fault is latched, (3) a health flag or score visible to the operator. Set `status` on every state transition; **a silent FAIL with no status update is FORBIDDEN**. Route logic faults to the operator via `BAlarmSourceExt` on a control-point child (alarm/fault/unacked → `BAlarmRecord` → `BAlarmService.routeAlarm` → console → ack). `[ev: corpus B808]` `[ev: corpus B827]`
- **Protection ownership** — the protection layer that RAISES a latch owns it: it sets the latch, holds the alarmed state, and waits for the operator reset. The SAFETY layer WATCHES (reads the latch) and does not raise. Never let two subsystems both write the same latch — that creates an un-auditable race. `[ev: corpus B805]`

## Protection anatomy `[ev: corpus B827]`

A protection is a latch (`freezeTripped`; or the CP-1 LP-floor shed, which is inline with NO named field — the silent case B824 flags) that OVERRIDES the normal command path. It has four tiers, and each
tier that exists must be VISIBLE — a protection that silently holds an output is indistinguishable from a broken relay
(`lint-silent-protection.sh`, `[ev: retro campaign9-silent-protection]`); the lint recognises BOTH patterns' surfaces — Pattern A: `BAlarmSourceExt` on a child control point; Pattern B: the B-adapter implementing `BIAlarmSource` + `newOffnormalAlarm`/`AlarmSupport`, identified via the `B<Pure>` naming pair. `[ev: retro campaign10-silent-protection-pattern-b]`):

1. **Setpoint + hysteresis** — the trip and the restart thresholds are two slots (`freezeSetpoint`/`freezeDiffStop`/`freezeDiffRestart`), never one. `[ev: corpus B821]`
2. **Latch** — a private boolean assigned in ONE pure function (`ColdRoomControl.freezeTrip(...)`), tested without Baja.
3. **Override** — the latch wins over HOA/mode at the apply stage (`valveInhibited()`), and the override is logged once per edge.
4. **Alarm** — the latch is SURFACED as an alarm record, one per edge, using one of two patterns:

   - **Pattern A — child point + `BAlarmSourceExt`** (when the latch can be expressed as a boolean point): declare a frozen
     child `BBooleanPoint` on the unit, hang a `BAlarmSourceExt` with `BBooleanChangeOfStateAlgorithm(alarmValue=true)` on
     it, and write the point's `out` from the latch in the recompute. The ext owns the edge (raises on false→true, clears on
     true→false) — the module keeps NO alarm state. Legality: the ext's PARENT must be a `BControlPoint`
     (`BPointExtension.isParentLegal` :64-66, narrowed by `BAlarmSourceExt.isParentLegal` :1073-1078) and the ALGORITHM's
     grandparent must be a `BBooleanPoint` (`BBooleanChangeOfStateAlgorithm.isGrandparentLegal` :86-89), so the ext cannot sit
     on the unit itself. `[ev: corpus B827 §827.3]`

   - **Pattern B — `BIAlarmSource` + transient `AlarmSupport`** (when the source is not a point, or several trips share one
     source): the component implements `BIAlarmSource` (a VISIBLE `@NiagaraAction BBoolean ackAlarm(BAlarmRecord)` whose
     `doAckAlarm` delegates to `support.ackAlarm` — never hidden: the console must invoke it; `Flags.HIDDEN` makes an action
     invisible in Workbench and unreachable over oBIX), creates `new AlarmSupport(this, alarmClass)` in `started()`, and
     calls `newOffnormalAlarm(alarmData)` / `toNormal(...)` ONLY on the edge computed by a pure edge machine
     (`AlarmEdge.decide(trip, nowOffnormal, recoveredPastDeadband) → FIRE|CLEAR|NONE`) that is re-seeded from the CURRENT
     condition in `started()` (a restart never re-fires). `[ev: corpus B827 §827.4 §827.6]`

   Both route a `BAlarmRecord` with `sourceState = offnormal`; high/low is an `alarmData` key, never a sourceState. The
   dashboard's alarm strip selects `sourceState = 'offnormal' or 'fault'`. The live routing is HARNESS-ONLY (station);
   WSL tests pin the structure (child declared, ext + algorithm present, drive line in the recompute; `implements
   BIAlarmSource`, `new AlarmSupport(` in `started()`, `newOffnormalAlarm` inside the FIRE branch) and the pure edge machine
   (pinned by the C9 alarm REDs qa/c9-alarm-cr3, qa/c9-alarm-cp1). `[ev: corpus B827 §827.3 §827.4]`

## kitControl patterns — exemplar-backed

- **LC1 / Writable priority array (16-level arbitration):** `WritableSupport.onExecute` scans levels 1→16 first-valid-wins; relinquish-default = ordinal 17 (`BPriorityLevel.fallback`), used only when all 16 are null; only levels 1 and 8 raise `OVERRIDDEN`; to make a config slot oBIX-writable add a NEW slot + link, never re-type the existing point. [ev: corpus B536]
- **LC2 / BLoopPoint PID:** ramp = rate-limiter + ramp-aware anti-windup; `executeTime` clamped [100ms, 60s]; `direct` action = cooling, `reverse` = heating; `disableAction` triggers bumpless pre-load; recommended tuning: `kP = (maxOut − minOut) / throttlingRange`; PI is recommended, PID seldom justified. [ev: corpus B539]
  → For third-party math/control library selection (Apache Commons Math, EJML, ND4J, MiniPID, state-machine libs) and the engine-thread/BWorker rule for heavy math: see `types/math-control-libraries.md`.
- **LC3 / Multi-input null contract:** nulls are SKIPPED, not zeroed (`BQuadMath.nonNullCount`, `BAnd.nullOnInactive`); latch = rising-edge vs latch-action both-edges; `switch`/`select` invalid → hold + invalid-flag. [ev: corpus B537]
- **LC4 / Control-logic fail-safe checklist (the 6 unsafe-unless-configured defaults):** `disableAction=hold` freezes the last command; `propagateFlags=0` lets a bad sensor drive the loop with no fault; `rampTime=0` = no anti-slam; alarm-ext is notification-only (no interlock); `999` sentinel is convention not framework; `clHVAC` strips Baja status. Operator recs: set `disableAction`/`Fallback` to safe posture, `propagateFlags=fault|stale|down`, `rampTime>0`, emergency L1 for interlocks. [ev: corpus B543]
- **LC5 / Point-extension chain (alarm + history):** `BAlarmSourceExt` → offnormal/fault algorithm (`BOutOfRange`/`FloatingLimit`/`TwoState`) + `AlarmService`; `BIntervalHistoryExt` (timer, default 15 min) vs `BCovHistoryExt` (COV `changeTolerance` deadband); alarm-ext NEVER writes the value/priority-array — it is notification-only. [ev: corpus B552]
- **LC6 / Scheduler seam DI:** instead of calling `Clock.schedule` directly in the arming method, inject a tiny interface `interface Sched { Object at(long delayMs); void cancel(Object t); }` — production wires a `Clock`-backed impl; a test wires a fake that records `at(delayMs)` calls. This makes "which path arms, with what delay, cancel-before-reschedule" unit-testable without a station. [ev: corpus B743 §743.3]

See also: `docs/how-to-create-coldroom-module.md` (end-to-end ColdRoomPan build); corpus **B729** (timer lifecycle: started/atSteadyState/clockChanged) + **B730** (rt idioms catalog).

→ For framework-extension authoring (SPIs, ORD schemes, point extensions, containers, queries, templates, jobs, watchdogs, action protection, minimal module): see `types/logic-authoring.md`.
- **A companion boolean/int flag set in the same method body as `Clock.schedule*` MUST be cleared in `stopped()` or `started()`, not only in the expiry handler:** if `stopped()` returns without clearing the flag, a disable→enable cycle (same object, no new instance) carries the stale `true` forward; the flag read after the re-arm is wrong. Cleared only in the expiry path does NOT count — clearing it in `stopped()` (or `started()`) is the only safe lifetime. [ev: corpus B801] [ev: corpus B812]
- **Schedule through `Clock.schedule*` only — never via `ScheduledExecutorService`, `Executors.*`, or `new Thread(...)` in a BComponent subclass:** the station `SecurityManager` denies `modifyThread` to module code; any JDK thread primitive throws at runtime and leaves the unit in a broken state. [ev: corpus B800 §800.3] [ev: corpus B806]
- **Guard every scheduling body reachable from `changed()` or `started()` (directly or one level deep) with `isRunning()|atSteadyState()`:** `changed()` fires during `activateLinks` before the engine fully accepts `Clock.schedule`; without a guard the schedule call throws `NotRunningException` and the timer is silently skipped for the whole session (observed ×6 in PANCCADIA logs). [ev: corpus B816]
- **Off-engine blocking I/O — use `BWorker`, never `Thread.start()`:** when a `BComponent` needs direct hardware/device IO (not via `BLink`/proxy), post `Runnable`s to a `BWorker` child component and deliver results back via `post()/postAsync()`; `Thread.start()` is wrong (unmanaged lifetime, no Baja fault propagation). Cascade failure mode: a blocking engine callback stalls Jetty worker threads that call `.get()` on `BComponent` state, saturating the Jetty pool and producing HTTP 503 (503-cascade). `[ev: corpus B730 §730.8 §G2]`

## Liveness watchdog recipe `[ev: corpus B812]`

An independent monitor on the producer's `lastTick` is the layer Tridium does NOT ship; the author must build it. This is timer defense-in-depth layer 4 (see §Safety fail-modes & timers above).

- **Producer side:** add a `TRANSIENT|SUMMARY|READONLY` `BAbsTime` slot `lastTick` to the producing component; set it to `Clock.now()` on every real execution cycle (NOT in the liveness monitor — the monitor only reads it). TRANSIENT = resets to NULL on every restart, which is the intended behavior (a fresh restart is not a stall).
- **Monitor side:** a separate scheduled or alarm-monitor component reads `producer.getLastTick()` at a configured cadence; computes `stall = now − lastTick`; triggers a `BStatus` fault and a `BAlarmRecord` when `stall > max(1, N) × period` (recommended N = 3; floor `max(1,...)` prevents divide-by-zero). Use `BAbstractAlarmMonitor` as the scaffold (periodic check, configurable interval, alarm edge-latch).
- **Why separate:** the monitor cannot sit in the same component it watches — a stuck engine thread leaves the monitor stuck too. A second component on its own cadence (or a `BAbstractAlarmMonitor`) remains independent.
- **Do not wait on wall-clock timers** in a WSL test — inject the `Sched` interface (see LC6 above) or test the pure stall-detection math (stall threshold, floor) off-station.

## Ship a tag dictionary `[ev: corpus B814]`

A module can auto-tag every component of its types with semantic tags using `BSmartTagDictionary` and `BNamespace`, so NEQL queries, navigation hierarchies, and Haystack tooling see the components with zero integrator effort.

- **`BSmartTagDictionary`** is the mechanism: declare a `@NiagaraType` subclass, register it as an agent under `TagDictionaryService`, and override `getRules()` to return `SmartTagRule[]` — each rule maps a Baja `Type` to a set of tag names/values. Components matching that type are auto-tagged on insert.
- **Constructor-seed API (alternative to `getRules`):** seed individual marker tags directly in the subclass constructor via `tagInfoList.add(SlotPath.escape(name), new BSimpleTagInfo(BMarker.DEFAULT))`, guarded by `if (get(name) == null)` for idempotency across re-runs. This is the Honeywell "seed in constructor" pattern; the `getRules()/SmartTagRule[]` pattern (B814) is the Haystack discovery path. Both extend `BSmartTagDictionary`; choose one per dictionary. `[ev: corpus B758 §758.1]`
- **Rule-based auto-tagging:** implement `BTagRule` + `BTagRuleCondition` to classify components automatically without per-instance annotation. The `getImpliedTag`/`addAllImpliedTags` engine in `BSmartTagDictionary` iterates `getTagRules()` and applies matching rules. Override `test(entity)` → `true/false` and return tag names/values from `getTags()`. `BEquipmentTypeTag` maps a folder's `displayName` to an equip-type tag via a `lookupTable` facet. `BIsPointProxyTypeRule` + `BIsPointProxyTypeCondition` match a driver point by its proxyExt `TYPE`. `[ev: corpus B758 §758.1]`
- **`BNamespace`** (optional): declare a named namespace in `module-include.xml` to own tag names that don't collide with the Haystack or Baja built-in namespaces.
- **Palette placement:** drag the dictionary instance under `TagDictionaryService` at commissioning (or auto-install it in `serviceStarted()`). NEQL queries that use `tag::` operators and the Hierarchy/Navigation view become addressable immediately.
- **Key rule:** tag names live in the lexicon (they are user-visible strings); keep them short, scope-stable, and consistent with the Haystack marker convention (no CamelCase in tag names).

### Our-modules tag recipe (namespace `angeles`) `[ev: corpus B750 §750.2, B758 §758.5]`

Concrete application of the dictionary for our cold-chain modules:

- **Namespace:** `angeles` (declare in `module-include.xml` as a `<namespace>` entry).
- **Marker tags:** `room`, `evaporator`, `compressor`, `defrost`.
- **Keying strategy:** one `BTagRule` per component TYPE (e.g. `entity instanceof BEvaporatorUnit`) — not by folder name, since our components are well-typed.
- **Auto-install:** place the dictionary instance under `TagDictionaryService` in `serviceStarted()` — no integrator drag required.
- **Result:** `station:|slot:/|neql:select * where tag::room` reaches all cold-room components with zero integrator effort after module deploy.
- **Deploy safety note:** this overlay is additive — no schema change, no containment change; deploy on an existing station is safe.

## BQL function reference card `[ev: corpus B1108]`

Three function families are available in BQL expressions. Resolution is reflective — names are not greppable from source; this card is the lookup surface.

**5 aggregate functions** (defined as `Type[]` fields on `BBqlLibrary`):
`count`, `sum`, `avg`, `min`, `max`

**7 expression scalar functions** (public static methods on `BBqlLibrary`):
`abs`, `ceil`, `floor`, `round`, `max`, `min`, `mod`

**24 `BBqlTime` library functions** (time/range helpers; call as `time.<fn>()` in a BQL expression):
`now`, `today`, `yesterday`, `thisWeek`, `lastWeek`, `thisMonth`, `lastMonth`, `thisQuarter`, `lastQuarter`, `thisYear`, `lastYear`, `startOf`, `endOf`, `add`, `subtract`, `between`, `before`, `after`, `atMidnight`, `atNoon`, `daysAgo`, `hoursAgo`, `minutesAgo`, `secondsAgo`

**Registering a custom BQL function:**
1. Declare a `public static` method on a `BObject` subclass (your library class).
2. Register it in `module-include.xml` via `moduleSpec::fnName()`.
3. Call it in BQL as `moduleName::fnName(args)`.

`[ev: retro module-hardening-reference-cards-deltas Δ3]`

## BFormat pattern reference `[ev: corpus B1112]`

`BFormat` is the display-name / annotation / history-label pattern language. Understand it before writing a format expression or processing user-supplied templates.

**Pattern syntax:**
- `%slot%` — resolve `slot` on the current context component via reflection
- `%a.b.c%` — chain access with `.` (traverse slots/methods left-to-right)
- `%slot?default%` — use literal `default` when `slot` resolves to null or throws
- `%time(<fmt>)%` — current date/time with a Java `SimpleDateFormat` pattern
- `%user%` — current user's display name
- `%substring(<slot>,<start>,<end>)%` — substring of a resolved slot value

**Reflection resolution order** (inside a `%…%` block):
1. Baja slot by name
2. Public zero-arg Java method (getter shape)
3. `ReflectionException` if neither resolves

**`FormatDenylist`:** N4 maintains a station-version-specific denylist that blocks known-dangerous reflection targets. The list limits the attack surface but does not eliminate it.

**SECURITY rule:** a `BFormat` pattern is reflective code execution. **NEVER construct a `BFormat` pattern from untrusted input** (user-supplied strings, BACnet device names, oBIX payloads). An attacker who controls the pattern string can traverse the component tree and read sensitive slots. Always author `BFormat` patterns at design time; treat them as code. `[ev: retro module-hardening-reference-cards-deltas Δ7]`
