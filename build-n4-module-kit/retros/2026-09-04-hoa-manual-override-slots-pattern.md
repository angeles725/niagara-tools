# Retro (PROPOSED delta — propose-never-apply) — the HOA manual-override slot pattern for a control component (Auto / On / Off per output)

- **Date**: 2026-09-04
- **Origin**: operator wanted per-compressor manual control (Auto / Encender / Apagar) on the CompPan rack, matching the evaporator HOA rows the dashboard already had for the rooms. Implemented `condenser1/2/3Mode` on `BCompressorControl` + the override logic in the pure `CompressorControl`. Built + verified (gate ALL PASS, 31/31 pure tests, getters confirmed in bytecode via `javap`).
- **Status**: PROPOSED. Documents a reusable authoring pattern (a HOA override slot wired into a pure-logic core) + the safe operator semantics. Adds no rule; follows the skill's step 6.

---

## Finding — how to add an operator HOA override to a control component

Three moving parts, in order:

**1. The slot (adapter, `B<Name>.java`).** One `@NiagaraProperty` per output, `type = "double"`, `defaultValue = "0d"`, `flags = Flags.SUMMARY | Flags.OPERATOR`. Encoding **0 = auto, 1 = on, 2 = off** — the exact ordinal the dashboard `hoaRow()` widget already writes (`{auto:"0", on:"1", off:"2"}`, see the dashboard write-surface retro). slotomatic generates the region + `getCondenserNMode()/setCondenserNMode(double)` from the annotation alone — no hand-written region. Prove it: `javap -p` on the built jar shows the `Property` + getter/setter.

**2. Pass it into the pure core, without breaking the existing tests.** The pure `step(...)` gains an `int[] modes` parameter. Keep the OLD signature as a thin **overload** that delegates with an all-AUTO array:
```java
void step(..., Cfg c) { step(..., AUTO_MODES, c); }        // back-compat: every existing test call still compiles
void step(..., int[] modes, Cfg c) { this.modes = modes; ... }
private static final int[] AUTO_MODES = { MODE_AUTO, MODE_AUTO, MODE_AUTO };
```
This kept all 28 prior tests unchanged; only the 3 new HOA tests use the modes overload. The adapter reads the slots and calls the modes overload: `int[] modes = { (int)Math.round(getCondenser1Mode()), ... }`. Add the mode slots to the `changed()` trigger list so a mode change re-runs the control.

**3. The override semantics (safe defaults, operator-confirmed).** Apply the override as the LAST step, after the auto staging + safety caps:
- **OFF** → `cmd[k]=false`, AND exclude from `available`/the rotation pick so the auto restages a *different* unit to still meet demand (a lockout, not a lost stage).
- **ON** → force `cmd[k]=true`, but **only if `(now - cmdSince[k]) >= minOff`** — a manual on must NOT short-cycle the motor. This is the one interlock ON keeps.
- **AUTO** → leave the auto command.
- Operator decision surfaced (leave it to them, don't assume): does a manual ON also respect the high-pressure / high-discharge cutout, or is it a full override (current: bypasses everything except min-off)? A "Hand" mode for maintenance typically overrides; the hardware cutout is the backstop. Flag it, let the operator choose before deploy.

## Why it matters

- The overload pattern is the cheap way to extend a pure decision core's signature without a test-file rewrite — reuse it for any new per-cycle input.
- Encoding the mode as the dashboard's existing `0/1/2` double means the servlet write path (`coerceValue` → `BDouble`) needs **zero** Java changes; the dashboard `hoaRow("...", "<panel>/condNMode", null, {stateOrd:"<panel>/condNState"})` wires straight to it.
- "ON respects min-off" is the non-negotiable safety line; everything else about Hand mode is an operator policy call, not a default to bake in silently.

## Cross-refs

- Dashboard HOA widget + write surface: `retros/2026-09-03-dashboard-servlet-write-surface-and-reader-authority.md`.
- `0 = disabled` vs `0 = block` for limits: `retros/2026-09-03-default-zero-limit-blocks-instead-of-disabling.md`.
- WSL pure tests: `retros/2026-09-04-junit-standalone-cached-jar-locations-for-wsl-pure-tests.md`.
