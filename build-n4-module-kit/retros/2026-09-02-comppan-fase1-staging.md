<!-- review-status: fresh · 2026-09-02 -->

# Retro — CompPan `-rt` FASE 1: rack de compresores (staging) — PROPOSED kit deltas

Status: **propose-never-apply**. A human folds these into `types/logic.md` / `METHODOLOGY.md`.
Context: a NEW pure-logic `-rt` module (3-compressor / 3-condenser rack staging) for PANCCADIA Leon, built from the ColdRoomPan pattern, designed + reviewed across three Claude sessions. Java-8 + slotomatic + jar, bytecode 52, signed, gate ALL PASS, 11/11 JUnit green.

## What this build PROVED (seeds for `types/logic.md`)

1. **The pure-class / Baja-adapter split scales to a STATEFUL controller, not just stateless functions.** ColdRoomPan's `ColdRoomControl` is stateless static methods; a rack needs runtime state (which compressors are on, per-compressor run-hours, min-on/min-off/stage timers, latched faults). Kept it testable by making `CompressorControl` a **stateful instance** (plain arrays, zero Baja) with one `step(now, inputs…, Cfg)` method; `BCompressorControl` reads slots → builds inputs → calls `step` → writes output slots. The instance holds transient state; only the run-HOURS are persisted (see #4). 11 JUnit tests drive `step` over advancing `now` and assert the arrays. This is the pattern to reach for whenever the control has memory.

2. **A monotonic-time sentinel needs a guard at EVERY use, not just one.** `long lastStageMs = Long.MIN_VALUE;` then `if ((now - lastStageMs) >= stageDelayMs)` — `now - Long.MIN_VALUE` **overflows to negative** for any real `Clock.millis()`, so the gate is false forever and **the rack never stages a compressor** (silent, catastrophic — nothing turns on). The author had already guarded the identical sentinel for `lastStepMs` (`== Long.MIN_VALUE ? 0`) but missed it on `lastStageMs`. Rule: a `Long.MIN_VALUE` "never happened" sentinel used in `(now - sentinel)` MUST be guarded `sentinel == Long.MIN_VALUE || (now - sentinel) >= …` at *every* call site. A second reviewer caught this before compile; it would have passed the gate and shipped dead.

3. **Sensor STATUS must gate the fault decision, or a dead sensor false-faults healthy equipment.** First cut: `running[k] = ampsValid[k] && amps[k] > threshold`, then "commanded ON + not running past prove-delay → fault + drop". An amp-sensor glitch (`ampsValid=false`) then reads as "failed to start" and **latches a healthy compressor out of the rack**. Fix: `running[k] = measured ? (amps>thr) : cmd[k]` (when you can't measure, assume the commanded state — never conclude failure from a dead sensor), and fault ONLY on a VALID low reading. Same status-first discipline as ColdRoomPan's `decideCall` (isNull/isValid before value). Extends METHODOLOGY: read status, and *distinguish "invalid reading" from "bad value"* — they drive opposite safe actions.

4. **Persist ONLY the state that must survive restart; reset the rest to a safe cold start.** Run-hours (for lead/lag rotation) live in non-transient `SUMMARY|READONLY` double slots, seeded into the pure instance ONCE at `atSteadyState` (a `hoursSeeded` flag), written back each cycle. Everything else (commands, running, faults, timers) is transient → a station restart is a clean "all off". Add a `resetTransient()` the adapter calls in `stopped()` so a component **disable→enable** (same instance reused, NOT a restart) also cold-starts — otherwise it resumes stale commands.

5. **A latched fault needs an operator-owned exit; never auto-retry a failed compressor.** Auto-retrying a compressor that failed to start can damage the motor. Latch the fault, expose a momentary `faultReset` slot (SUMMARY|OPERATOR) that the adapter maps to `ctl.resetFaults()` and self-resets to false. A genuinely failed unit stays out (with alarm) until the operator acknowledges. Mirrors the "safety posture is an operator slot" lesson from rt-hardening #4.

6. **Two cheap amperage alarms that cost almost nothing:** proof-of-run (CMD ON + no amps past prove-delay = failed start) AND its inverse **stuck-contactor** (CMD OFF + amps present = compressor did not stop, critical). Both fall out of the same `running[k]`/`cmd[k]` pair. Easy to write only the first and forget the second.

## Lifecycle / adapter (seeds for `types/logic.md`)

7. **Arm the periodic heartbeat INDEPENDENT of the first `execute()`.** Hour integration + prove-timeouts need time to pass even when no slot changes, so a `Clock.schedule` tick drives `execute()` every 5 s (copy `BDefrostController`'s ticket idiom: schedule → re-arm in the callback only `if (isRunning())` → cancel in `stopped()`, single ticket, never leaks). BUT if `armTick()` runs *after* `execute()` inside one try/catch and the first `execute()` throws, the tick is skipped and the control degrades to reactive-only forever. Arm the tick in its OWN try BEFORE the first cycle.

8. **No-reentrancy comes free IF `execute()` writes only output slots and `changed()` filters to input+config slots only.** Verified: the write→changed→execute loop cannot form because none of the outputs (`condenser*`, `*Running/Fault/Hours`, `demand`, `stagesOn`, alarms) are in `changed()`'s filter. This is the structural fix for the BRoomPanel flood bug, stated as an invariant: outputs ∉ changed-filter.

## Process (seeds for METHODOLOGY / BUILD-LOOP)

9. **Adversarial second-reviewer BEFORE compile pays for itself on control logic.** The `lastStageMs` overflow (#2) and the sensor false-fault (#3) were both caught by a separate reviewer reading only the pure class — they compile and pass the gate cleanly; only behavior is wrong. A gate (bytecode 52 / signed / types resolve) does NOT catch control-logic defects. For a logic module, add a pure-logic review pass + JUnit that encodes INTENDED behavior (so it fails on the real defect) as an explicit loop step.

10. **`niagaraTest` does not run in WSL — run the pure JUnit standalone.** The module's own `niagaraTest` gradle task needs the native bin/test harness (Windows) + a dev license. The pure class has zero Baja, so `javac -cp junit-4.13.2.jar` + `java org.junit.runner.JUnitCore` runs the suite in WSL (junit/hamcrest jars are already in `~/.gradle/caches`). Build the module with `:<MOD>-rt:jar` (not the test task). This is another reason to keep ALL control logic Baja-free.

11. **A brand-new module's plugin auto-install does NOT need the station stopped** (unlike an update): its jar is not yet in `niagara_home/modules`, so there is no locked file to overwrite. `:CompPan-rt:clean :CompPan-rt:jar` built green against the live 4.14 install with the station running.
