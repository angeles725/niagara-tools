# Spec: silent-protection Pattern B surface (S23)

**Capability**: `module-silent-protection-lint` — surface recogniser
**PR**: PR3 (`feat/c10-silent-protection-pattern-b`)
**QA RED**: `qa/c10-silent-protection-surfaces` tip `f981754` (base `df8c7ec`) `[CERT]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K14/K19/K20/D9b/comment-strip/observed-flip/real-tree-smoke)
**Change class**: FP-only — removes a false WARN; Pattern A and all existing exemptions unchanged.

---

## Delta — MODIFIED requirements

The following MODIFY the surface recogniser in `lint-silent-protection.sh` (section B at `:220-226` / criterion (1) at `:424` @ `cb79676`). All other patterns (P1, P3, P4, P7), exemptions (private-field, effect-slot), dedupe, and exit codes are UNCHANGED.

| ID | Requirement | ev |
|----|-------------|-----|
| R-S23.1 | The surface recogniser MUST accept **Pattern B**: a trip in a class that `implements … BIAlarmSource` AND whose source contains `new AlarmSupport(` or `newOffnormalAlarm(` has a tier-1 alarm surface and MUST NOT produce a WARN for any trip in that class. | `[ev: B831 §831.3]` `[ev: apply-package S23 2f710626d]` `[ev: lint-silent-protection.sh:222,:424 @ cb79676]` |
| R-S23.2 | The **adapter→pure follow**: a trip in a pure class `C` (no `B` prefix, no `BIAlarmSource`) is considered surfaced when the source tree contains an adapter class `B<C>` (by the `B<PureClass>` naming convention) that carries a Pattern B surface (`implements BIAlarmSource` + `new AlarmSupport(` or `newOffnormalAlarm(`). | `[ev: B831 §831.3]` `[ev: apply-package S23 2f710626d §Refinement]` |
| R-S23.3 | The adapter→pure follow is **convention-based** (`B<PureClass>` naming pair). When the convention is not followed, the trip is over-reported (a WARN) rather than silenced. This stated limitation is documented in the script's doctrine row. | `[ev: proposal RK6]` |
| R-S23.4 | **Pattern A** (a file containing `BAlarmSourceExt` or `BAlarmRecord`, `:222/:424`) is **unchanged** — it continues to exempt trips in files that carry Pattern A tokens. | `[ev: apply-package S23 2f710626d]` `[ev: lint-silent-protection.sh:222 @ cb79676]` |
| R-S23.5 | All existing exemptions — private-field, effect-slot, per-trip dedupe on `<file>:<line>` — are **unchanged**. | `[ev: apply-package S23 2f710626d]` |
| R-S23.6 | SP-smoke **CONTRACT** after fix: `lint-silent-protection.sh` on `CompPan-rt/src` at client `ff1b659` MUST emit **0 WARN**; `CompressorControl.java:294` MUST be **ABSENT** from output. | `[ev: B831 §831.3]` `[ev: qa/c10-silent-protection-surfaces f981754 S23-pos]` |
| R-S23.7 | `lint-silent-protection.sh` on `ColdRoomPan-rt/src` at client `ff1b659` MUST emit **0 WARN** (CR-3 recognised via Pattern A `freezeAlarmPt`/`BAlarmSourceExt` — absence of WARN proves Pattern A did not regress). | `[ev: apply-package S23 2f710626d §Expected smoke]` |
| R-S23.8 | DashboardPan-rt and DashboardPan-ux MUST produce **0 WARN** each. | `[ev: apply-package S23 2f710626d §Expected smoke]` |
| R-S23.9 | Row grammar and exit codes (0 WARN-only, 1 any FAIL, 3 usage/env) are **unchanged**. | `[ev: lint-silent-protection.sh:19 @ cb79676]` `[ev: K20]` |

---

## Scenarios

### S23-pos (must-pass — Pattern B adapter surface recognised)

**Given** a fixture containing:
- A pure class `CompressorControl` with a trip: `target = Math.min(target, onCount - 1)` inside an `if (suction < c.suctionLowLimit)` guard;
- An adapter class `BCompressorControl` in the same source tree that contains `implements BIAlarmSource`, `new AlarmSupport(this, …)`, and `alarm.newOffnormalAlarm(…)`.

**When** `lint-silent-protection.sh` runs (comments stripped before matching).

**Then** **0 WARN** for that trip. The fixture MUST include a comment-only decoy `// BIAlarmSource` that does NOT satisfy the Pattern B check.

`[ev: qa/c10-silent-protection-surfaces f981754 S23-pos]` `[ev: B831 §831.3 — BCompressorControl.java:447,:1882,:2093 @ ff1b659]`

---

### S23-neg (regression guard — no surface → WARN stays)

**Given** a fixture containing:
- A pure class with the same trip pattern (Math.min(target, onCount-1));
- NO adapter class with `BIAlarmSource` / `AlarmSupport` / `newOffnormalAlarm`;
- NO `BAlarmSourceExt` or `BAlarmRecord` in scope.

**When** `lint-silent-protection.sh` runs.

**Then** **1 WARN** row emitted for that trip.

`[ev: qa/c10-silent-protection-surfaces f981754 S23-neg]`

---

### SP-smoke-comppan (real-tree — CompPan-rt)

**Given** `CompPan-rt/src` at client `ff1b659` (worktree `Leon-Guanjuato-worktrees/main-ff1b659`).

**When** `lint-silent-protection.sh` runs.

**Then** exit **0**, **0 WARN**, `CompressorControl.java:294` **ABSENT** from output.

`[ev: qa/c10-silent-protection-surfaces f981754 S23-pos]` `[ev: B831 §831.3]`

---

### SP-smoke-coldroom (real-tree — Pattern A regression guard)

**Given** `ColdRoomPan-rt/src` at client `ff1b659`.

**When** `lint-silent-protection.sh` runs.

**Then** **0 WARN** (CR-3 freeze trip still recognised via Pattern A `BAlarmSourceExt`; absence of WARN proves Pattern A did not regress).

`[ev: apply-package S23 2f710626d §Expected smoke]`

---

### SP-smoke-dashboard (real-tree — DashboardPan both profiles)

**Given** `DashboardPan-rt/src` and `DashboardPan-ux/src` at client `ff1b659`.

**When** `lint-silent-protection.sh` runs on each.

**Then** **0 WARN** on each.

`[ev: apply-package S23 2f710626d §Expected smoke]`

---

### S23-observed-flip

**Given** the S23-neg fixture with the Pattern B recogniser applied.

**When** the `BIAlarmSource` check is removed from the recogniser.

**Then** S23-neg flips from **1 WARN → 0 WARN** (OBSERVED, verbatim output captured).

`[ev: proposal SC-7]` `[ev: K13]`

---

## Success criteria (this capability)

- [ ] `lint-silent-protection.sh` on CompPan-rt @ `ff1b659` emits **0 WARN**; `CompressorControl.java:294` **ABSENT**.
- [ ] ColdRoomPan-rt **0** (Pattern A CR-3 still recognised — absence pin proves no Pattern A regression).
- [ ] DashboardPan **0**.
- [ ] S23-neg (trip, no surface) still WARNs with an OBSERVED flip.
- [ ] Comment-decoy pin: `// BIAlarmSource` does NOT satisfy the Pattern B check.
- [ ] `kit-links.bats` green; `shellcheck 0.10.0` exits 0; 0 attribution trailers.
