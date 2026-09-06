# Spec: lint-timers companion-flag scope (S21)

**Capability**: `module-timer-lint` — `companion-flag` check
**PR**: PR1 (`feat/c10-lint-timers-scope`)
**QA RED**: `qa/c10-lint-timers-fp` tip `52ebd11` (base `df8c7ec`) `[CERT]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K14/K19/K20/D9b/comment-strip/observed-flip/real-tree-smoke)
**Issue**: kit issue #89 (closed by this PR)

---

## Delta — MODIFIED requirements

The following MODIFY the `companion-flag` sub-check within `lint-timers.sh`. All other checks (`timer-ticket`, `discarded-ticket`, `jdk-thread`, `changed-sched`) and their row grammar and exit codes are UNCHANGED.

| ID | Requirement | ev |
|----|-------------|-----|
| R-S21.1 | A companion-flag `FAIL` MUST require BOTH guards: (1) the flag identifier is a **class-FIELD** — its declaration appears at class scope (`private/protected/public/static … boolean/int <name>`) outside any method body; (2) the `Clock.schedule*` call appears in the **same enclosing method body** as the `flag = true` assignment. Either guard alone failing clears the FAIL. | `[ev: B831 §831.1]` `[ev: apply-package S21 2f710626d]` |
| R-S21.2 | A **method-local** boolean (`type name = …;` declared inside a method body) MUST NEVER produce a companion-flag `FAIL`, regardless of whether `Clock.schedule*` calls appear elsewhere in the same file. | `[ev: B831 §831.1]` `[ev: qa/c10-lint-timers-fp 52ebd11 S21-neg]` |
| R-S21.3 | A `Clock.schedule*` call in a **different method body** from the `= true` assignment MUST NEVER pair with that assignment to produce a companion-flag `FAIL`. | `[ev: B831 §831.1]` |
| R-S21.4 | The class-field detection pass MUST be added before Pass 1. It collects field names whose declaration matches a class-scope field pattern (not inside method braces). Pass 1 MUST filter: an identifier that is not in the collected field set is silently skipped. | `[ev: apply-package S21 2f710626d §Refinement]` `[ev: lint-timers.sh:135-212 @ cb79676]` |
| R-S21.5 | Row grammar for companion-flag `FAIL` is **unchanged**: `FAIL  companion-flag  <file>: flag '<name>' set beside Clock.schedule* not cleared in stopped()/started()`. | `[ev: lint-timers.sh:212 @ cb79676]` |
| R-S21.6 | Exit codes are **unchanged**: 0 = no `FAIL` · 1 = any `FAIL` · 2 = usage (note: usage exit is 2 in lint-timers.sh, consistent with its header contract — K20 disjoint within this script). | `[ev: lint-timers.sh:44 @ cb79676]` `[ev: K20]` |

---

## Scenarios

### S21-neg (must-pass after fix)

**Given** a fixture Java file containing:
- A method-local declaration `boolean anyNoHardware = false;` inside a method body that contains NO `Clock.schedule*` call;
- `Clock.schedule*` calls in OTHER method bodies;
- The ticket is cancelled in `stopped()`.

**When** `lint-timers.sh` runs on that fixture (with `//` and `/* */` stripped before matching).

**Then** exits 0 and NO `companion-flag` row appears for `anyNoHardware`. The fixture MUST include a comment-only decoy `// anyNoHardware = true` that does NOT satisfy the assertion.

`[ev: qa/c10-lint-timers-fp 52ebd11 S21-neg]` `[ev: B831 §831.1 — BDefrostController.java:718 method-local @ ff1b659]`

---

### S21-pos (regression guard — must-fail after fix)

**Given** a fixture Java file containing:
- A class-FIELD `private boolean startingUp;`;
- A method body that sets `startingUp = true` AND calls `Clock.schedule*()`;
- `stopped()` does NOT contain `startingUp = false`.

**When** `lint-timers.sh` runs on that fixture.

**Then** exits 1 with a `companion-flag` `FAIL` row naming `startingUp`.

`[ev: qa/c10-lint-timers-fp 52ebd11 S21-pos]` `[ev: B831 §831.1 — BCompressorControl.java:1760/1764 real shape @ ff1b659]`

---

### S21-smoke-coldroom (real-tree smoke)

**Given** `ColdRoomPan-rt/src` at client `ff1b659` (worktree `Leon-Guanjuato-worktrees/main-ff1b659`).

**When** `lint-timers.sh` runs.

**Then** exit **0** and `anyNoHardware` is **ABSENT** from all output. Exact FAIL count = **0** for companion-flag.

`[ev: qa/c10-lint-timers-fp 52ebd11 S21-smoke]` `[ev: apply-package S21 2f710626d §Expected smoke]`

---

### S21-smoke-no-regression (real-tree smoke — four roots)

**Given** all four client module roots (ColdRoomPan-rt, CompPan-rt, DashboardPan-rt, DashboardPan-ux) at `ff1b659`.

**When** `lint-timers.sh` runs on each.

**Then** the verdict for CompPan-rt, DashboardPan-rt, DashboardPan-ux is **UNCHANGED** from the pre-fix run (no new FAILs introduced, no existing FAILs removed except those that are genuine false positives addressed by S21).

`[ev: apply-package S21 2f710626d §Expected smoke]` `[ev: METHODOLOGY.md K22]`

---

### S21-observed-flip

**Given** the S21-neg fixture with the class-field guard applied.

**When** the class-field guard is reverted (removing guard 1 only).

**Then** `S21-neg` flips from exit 0 → exit 1 (OBSERVED, verbatim output captured).

`[ev: proposal SC-7]` `[ev: K13]`

---

## Success criteria (this capability)

- [ ] `lint-timers.sh` on `ColdRoomPan-rt/src` @ `ff1b659` exits **0**; `anyNoHardware` **ABSENT**.
- [ ] `S21-pos` (class-FIELD, same-method, never cleared) **still FAILs** with an OBSERVED flip.
- [ ] CompPan-rt / DashboardPan rt/ux verdicts **unchanged**.
- [ ] kit issue **#89 closed** by this PR.
- [ ] Comment-decoy pin: `// anyNoHardware = true` does NOT satisfy S21-neg.
- [ ] `kit-links.bats` green; `shellcheck 0.10.0` exits 0; 0 attribution trailers.
