# Spec: ext-writable-shape per-slot writing action (S22)

**Capability**: `module-ext-writable-shape-lint` — exemption rule
**PR**: PR2 (`feat/c10-ext-writable-per-slot`)
**QA RED**: `qa/c10-ext-writable-per-slot` tip `954ebd7` (pins-only successor of `00e31ae`; adds EW-s22-neg2) (base `df8c7ec`) `[CERT]` — re-read the tip at apply.
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K14/K19/K20/D9b/comment-strip/observed-flip/real-tree-smoke)
**Contract change**: EW10 CompPan-rt re-pins from **0** to **1** WARN (subject `faultReset`).

---

## Delta — MODIFIED requirements

The following MODIFY the exemption logic in `lint-ext-writable-shape.sh` (Pass 1, class-level `has_action` at `:82-91` @ `cb79676`). Pass 2 (property scan), row grammar, and exit codes are UNCHANGED.

| ID | Requirement | ev |
|----|-------------|-----|
| R-S22.1 | The class-level `has_action` exemption (`lint-ext-writable-shape.sh:82-91`) MUST be replaced by a **per-slot writing-action** check: a complex OPERATOR slot `X` is exempt from WARN **only** when the body of an `@NiagaraAction` whose annotation `name` attribute maps to the `do<Action>()` handler method contains a write to slot `X` via `setX(`, `getX().setValue(`, or `.set(<Xprop>,`. | `[ev: B831 §831.2]` `[ev: apply-package S22 2f710626d]` `[ev: lint-ext-writable-shape.sh:82-91 @ cb79676]` |
| R-S22.2 | The action-name → handler mapping MUST follow the Niagara `do<Action>` convention: an `@NiagaraAction(name="ackAlarm")` annotation maps to the method `doAckAlarm()`. The scan of writes to `X` MUST be restricted to THAT handler's body. | `[ev: B831-G1]` `[ev: investigador1 685e7981e — new action-aware pass over section-D method-boundary parser :250-306]` |
| R-S22.3 | `execute()`, `changed()`, and generated setters (setX() calls inside the component lifecycle loop) MUST be **excluded** from the action-body scope filter. A write to `X` inside these methods does NOT exempt slot `X`. | `[ev: B831 §831.2 — setFaultReset(false) at BCompressorControl.java:2025 inside execute/changed is NOT exempt]` `[ev: investigador1 685e7981e]` |
| R-S22.4 | An `@NiagaraAction` body that writes a **different** slot `Y` (not `X`) MUST NOT exempt slot `X`. | `[ev: B831-G1]` `[ev: qa/c10-ext-writable-per-slot 954ebd7 EW-s22-neg2]` |
| R-S22.5 **(CONTRACT CHANGE — EW10)** | `lint-ext-writable-shape.sh` on `CompPan-rt/src` at client `ff1b659` MUST emit **exactly 1 WARN**, with subject `faultReset` asserted **by name** (not by line number — the lint anchors the `@NiagaraProperty(` open at `BCompressorControl.java:381`). No other CompPan-rt slot MUST appear in the WARN output (absence pin). | `[ev: B831 §831.2]` `[ev: qa/c10-ext-writable-per-slot 954ebd7 EW10]` `[ev: QA 2026-09-07 max-WARN run on ff1b659]` |
| R-S22.6 | ColdRoomPan-rt MUST produce **0 WARN**; DashboardPan-rt MUST produce **exactly 1 WARN** (`BRoomPanel.setpoint` — unchanged from C9); DashboardPan-ux MUST produce **0 WARN**. | `[ev: apply-package S22 2f710626d §Expected smoke]` |
| R-S22.7 | Row grammar is **unchanged**: `WARN  ext-writable-shape  <file>:<line>  <slot>: OPERATOR <type> with no writing action …` | `[ev: lint-ext-writable-shape.sh:182-186 @ cb79676]` |
| R-S22.8 | Exit codes are **unchanged**: 0 = WARN-only (no `--strict`); 1 = any WARN under `--strict`; 3 = usage/env. `[ev: K20]` | `[ev: lint-ext-writable-shape.sh:194 @ cb79676]` |

---

## Scenarios

### EW-s22-pos (must-pass — action body writes the slot)

**Given** a fixture Java class containing:
- A complex OPERATOR slot `setpoint` (`BStatusNumeric`, flags `OPERATOR`);
- An `@NiagaraAction(name="bumpSetpoint")` whose handler `doBumpSetpoint()` body contains `setSetpoint(…)`.

**When** `lint-ext-writable-shape.sh` runs on that fixture (comments stripped before matching).

**Then** **0 WARN** for `setpoint`. The fixture MUST include a comment-only decoy `// @NiagaraAction` that does NOT satisfy the action check.

`[ev: qa/c10-ext-writable-per-slot 954ebd7 EW-s22-pos]` `[ev: B831 §831.2]`

---

### EW-s22-neg (must-warn — unrelated action does not exempt)

**Given** a fixture Java class containing:
- A complex OPERATOR slot `faultReset` (`BStatusBoolean`, flags `SUMMARY|OPERATOR`);
- `@NiagaraAction` entries (`tick`, `powerOnExpired`) whose bodies do NOT write `faultReset`.

**When** `lint-ext-writable-shape.sh` runs on that fixture.

**Then** **1 WARN** with subject `faultReset` by name.

`[ev: qa/c10-ext-writable-per-slot 954ebd7 EW-s22-neg]` `[ev: B831 §831.2]`

---

### EW-s22-neg2 — B831-G1 pin (different slot does not exempt)

**Given** a fixture Java class containing:
- A complex OPERATOR slot `faultReset`;
- An `@NiagaraAction(name="ackAlarm")` whose handler `doAckAlarm()` body writes an alarm acknowledgement field (`ackAlarm` or similar), but NOT `faultReset`.

**When** `lint-ext-writable-shape.sh` runs.

**Then** `faultReset` is **NOT exempt** and **1 WARN** appears for it.

`[ev: B831-G1]` `[ev: qa/c10-ext-writable-per-slot 954ebd7 EW-s22-neg2]`

---

### EW10-smoke (real-tree — CompPan-rt)

**Given** `CompPan-rt/src` at client `ff1b659` (worktree `Leon-Guanjuato-worktrees/main-ff1b659`).

**When** `lint-ext-writable-shape.sh` runs.

**Then** exactly **1 WARN**, subject **`faultReset`** (by name), **no other CompPan-rt slot** appears in output (absence pin).

`[ev: qa/c10-ext-writable-per-slot 954ebd7 EW10]` `[ev: B831 §831.2]`

---

### EW-smoke-all (real-tree — four roots)

**Given** all four client module roots at `ff1b659`.

**When** `lint-ext-writable-shape.sh` runs on each.

**Then**:
- ColdRoomPan-rt: **0 WARN**
- CompPan-rt: **1 WARN** (subject `faultReset`)
- DashboardPan-rt: **1 WARN** (subject `BRoomPanel.setpoint`)
- DashboardPan-ux: **0 WARN**

`[ev: apply-package S22 2f710626d §Expected smoke]` `[ev: METHODOLOGY.md K22]`

---

### S22-observed-flip

**Given** the EW-s22-neg fixture with the per-slot body-follow applied.

**When** the per-slot check is reverted to class-level `has_action`.

**Then** EW-s22-neg flips from **1 WARN → 0 WARN** (OBSERVED, verbatim output captured).

`[ev: proposal SC-7]` `[ev: K13]`

---

## Success criteria (this capability)

- [ ] `lint-ext-writable-shape.sh` on CompPan-rt @ `ff1b659` emits exactly **1 WARN**, subject `faultReset` **by name**, no other slot flips (absence pin).
- [ ] ColdRoomPan-rt **0**, DashboardPan-rt **1** (`BRoomPanel.setpoint` unchanged), DashboardPan-ux **0**.
- [ ] B831-G1 pin holds: `doAckAlarm` writing a different slot does NOT exempt `faultReset`.
- [ ] OBSERVED flip on EW-s22-neg.
- [ ] Comment-decoy pin: `// @NiagaraAction` does NOT satisfy the action check.
- [ ] `kit-links.bats` green; `shellcheck 0.10.0` exits 0; 0 attribution trailers.
