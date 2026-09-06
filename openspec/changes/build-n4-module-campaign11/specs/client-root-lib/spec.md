# Spec: client-root lib — centralise client-tree defaults (T2)

**Capability**: `kit-test-harness` — `tests/lib/client-root.bash`
**PR**: PR3 (`feat/c11-client-root-lib`)
**QA RED**: `qa/c11-client-root` tip `54078f6` (base `dab0807`) — `tests/lib/client-root.bash` existence + no absolute `Leon-Guanjuato` literal in `tests/*.bats` outside the lib. The commit body notes: LD5 flips to exit 0/clean on `main-ff1b659`; RC8 unchanged (1 FAIL). `[ev: explore §3.2; proposal PR3 row]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K19/K20/K21/K22/K24(7)/D9b/comment-strip/observed-flip/real-tree-smoke/smoke-assertion-class/fragment-merge)
**Affected tests (10 sites)**: `tests/{lint-timers,lint-silent-protection,ext-writable-shape,demand-in-scope,lint-write-path,lint-delays,rc-scan,c8-close,c9-close,c10-close}.bats`

---

## Delta — NEW and MODIFIED requirements

### New library — `tests/lib/client-root.bash`

| ID | Requirement | ev |
|----|-------------|-----|
| R-T2.1 | A new file `tests/lib/client-root.bash` MUST exist. It owns the **ONE** default client read root for the entire test suite. The default resolves to the `Leon-Guanjuato-worktrees/main-ff1b659` worktree (commit `ff1b659`). | `[ev: apply-package 0ad09c658]` `[ev: QA 54078f6 C11-T2-lib-exists]` |
| R-T2.2 | The library MUST export three variables: `C9_CLIENT_ROOT`, `C9_CLIENT_REPO`, and `C8_CLIENT_REPO`, all resolving to paths within the `main-ff1b659` worktree by default. No bats file outside `tests/lib/client-root.bash` may hardcode a path that includes `Leon-Guanjuato` or `Cliente/Leon-Guanjuato`. | `[ev: apply-package 0ad09c658]` `[ev: proposal R3 — three exported names]` |
| R-T2.3 | **Environment override wins**: if `C9_CLIENT_ROOT`, `C9_CLIENT_REPO`, or `C8_CLIENT_REPO` is already set in the environment before the library is sourced, the library MUST NOT overwrite it. The env value takes precedence over the built-in default. Each of the three variables is independently overridable. | `[ev: apply-package 0ad09c658]` `[ev: proposal SC-6 — env override wins]` |

### Modified — absolute path elimination (10 sites)

| ID | Requirement | ev |
|----|-------------|-----|
| R-T2.4 | After PR3 merges, the count of absolute `Leon-Guanjuato` path literals in `tests/*.bats` OUTSIDE `tests/lib/client-root.bash` MUST be **exactly 0**. The 10 sites at `dab0807` are: `tests/ext-writable-shape.bats:26` (`C9_CLIENT_ROOT`), `tests/demand-in-scope.bats:27` (`C9_CLIENT_ROOT`), `tests/lint-silent-protection.bats:30` (`C9_CLIENT_ROOT`), `tests/lint-timers.bats:418` (`C9_CLIENT_ROOT`), `tests/lint-write-path.bats:338` (`C9_CLIENT_ROOT`), `tests/c9-close.bats:108` (`C9_CLIENT_REPO`), `tests/c10-close.bats:90` (`C9_CLIENT_REPO`), `tests/c8-close.bats:107` (`C8_CLIENT_REPO`), `tests/lint-delays.bats:53` (bare `$HOME/…` path — no override form), `tests/rc-scan.bats:75` (bare `$HOME/…` path — no override form). All 10 MUST be converted. | `[ev: apply-package 0ad09c658]` `[ev: QA 54078f6 C11-T2-no-hardcode]` `[ev: proposal R3 — 10 sites named]` |
| R-T2.5 | `tests/lint-delays.bats` (LD5) and `tests/rc-scan.bats` (RC8) MUST gain the env-override form they currently lack: each MUST source `tests/lib/client-root.bash` (or equivalent sourcing) and use `C9_CLIENT_ROOT` (or the appropriate exported variable) for the real-tree path. No bare `$HOME/…/Leon-Guanjuato/…` literal may remain in either file. | `[ev: apply-package 0ad09c658]` `[ev: QA 54078f6 C11-T2-no-hardcode]` |

### Modified — retargeted tail smokes

| ID | Requirement | ev |
|----|-------------|-----|
| R-T2.6 | **LD5 retarget**: after PR3, `tests/lint-delays.bats` LD5 MUST assert that `lint-delays.sh` on the `ColdRoomPan-rt/src` at `main-ff1b659` (worktree `Leon-Guanjuato-worktrees/main-ff1b659`, commit `ff1b659`) exits **0** with no FAIL rows. The ColdRoomPan defrost `time <= 0` bug was fixed after C9 and is no longer present at `ff1b659`. The assertion `exit 1 + FAIL BDefrostController` (true only on the stale `4f5f1c7`) MUST NOT appear. | `[ev: proposal §Intent ¶4]` `[ev: lead re-measure 2026-09-07 on main-ff1b659]` `[ev: investigador1 2nd read 6f0069155]` `[ev: ../cross-cutting.md — smoke-assertion-class]` |
| R-T2.7 | The delay-floor **rule** MUST remain pinned by the existing synthetic fixtures `LD1`, `LD3`, and `LD6`. No new synthetic fixture is owed for the delay-floor rule because of the LD5 retarget. LD1/LD3/LD6 already assert exit 1 + FAIL for zero-floor patterns on synthetic inputs; LD5 retargeted to `ff1b659` pins the real-tree current clean state. | `[ev: proposal SC-7]` `[ev: proposal §Alternatives — LD5-class rot]` |
| R-T2.8 | **c8-close SC1-smoke retarget**: after PR3, the `c8-close.bats` SC1-smoke (which reads `C8_CLIENT_REPO`) MUST be re-pinned to `main-ff1b659`. On `ff1b659` the relevant smoke asserts the current clean state. | `[ev: apply-package 0ad09c658]` `[ev: proposal SC-7]` |
| R-T2.9 | **RC8 unchanged**: after PR3, `tests/rc-scan.bats` RC8 MUST continue to assert exit **1** with a `FAIL` row naming `host` for the DashboardPan-ux rc tree. The `:701` host literal in `DashboardPan-ux/src/rc/index.html` is present at `ff1b659` — the verdict does not change when the path is retargeted from `4f5f1c7` to `ff1b659`. | `[ev: proposal SC-7]` `[ev: lead re-measure 2026-09-07 — RC8 is 1 FAIL on both trees]` |

### No real-tree smoke may assert FAIL without a named ticket

| ID | Requirement | ev |
|----|-------------|-----|
| R-T2.10 | After PR3, **no** real-tree smoke in `tests/*.bats` MAY assert a FAIL (exit 1 or a FAIL row) without either: (a) the rule being separately covered by at least one synthetic fixture that always flags that pattern (and the smoke therefore asserts the current clean state), or (b) a named issue ticket documenting the known defect the smoke pins. RC8 satisfies (b) implicitly — the host literal is a documented known issue, not an accidentally failing rule. All retargeted smokes MUST be verified to satisfy (a) or (b) at apply time. | `[ev: ../cross-cutting.md — smoke-assertion-class]` `[ev: proposal RK5]` |

---

## Scenarios

### C11-T2-lib-exists (RED — lib file must exist)

**Given** the test suite at `dab0807`.

**When** `tests/lib/client-root.bash` is sourced from within any bats file.

**Then** the file **exists** and sources without error. On `dab0807` the file does not exist (C11-T2-lib-exists is RED — OBSERVED). `[ev: QA 54078f6]`

---

### C11-T2-no-hardcode (RED — 10 → 0 absolute literals)

**Given** all `tests/*.bats` files at `dab0807`.

**When** `grep -r "Leon-Guanjuato" tests/*.bats` is run (excluding `tests/lib/client-root.bash`).

**Then** **10 matches** are found (RED on `dab0807` — OBSERVED). After PR3 the count is **0**.

`[ev: QA 54078f6 C11-T2-no-hardcode]` `[ev: proposal R3 — 10 sites named]`

---

### T2-env-unset-suite-green (suite runs with all three variables unset)

**Given** `C9_CLIENT_ROOT`, `C9_CLIENT_REPO`, and `C8_CLIENT_REPO` all **unset** in the environment.

**When** the full bats suite runs against the kit.

**Then** the suite exits **0** (no failures caused by missing overrides). The library's built-in default resolves the paths correctly.

`[ev: proposal SC-6]` `[ev: R-T2.3]`

---

### T2-env-override-wins (one pin per variable)

**Given** `C9_CLIENT_ROOT` set to an arbitrary writable directory in the test environment (e.g. `$BATS_TEST_TMPDIR/fake-root`).

**When** `tests/lib/client-root.bash` is sourced and a lint is invoked using `C9_CLIENT_ROOT`.

**Then** the lint runs against `$BATS_TEST_TMPDIR/fake-root`, NOT the built-in default path. The env value wins.

`[ev: R-T2.3]` `[ev: proposal SC-6 — env override wins]`

The same pin is required for `C9_CLIENT_REPO` and `C8_CLIENT_REPO` independently.

---

### LD5-clean (LD5 retargeted to ff1b659 — exit 0)

**Given** `ColdRoomPan-rt/src` at `Leon-Guanjuato-worktrees/main-ff1b659` (`ff1b659`). `[ev: K21]`

**When** `lint-delays.sh` runs on that tree (via the `C9_CLIENT_ROOT` path from `client-root.bash`).

**Then** exits **0** with **0 FAIL rows**. The ColdRoomPan defrost `time <= 0` bug (that produced `FAIL BDefrostController` on the stale `4f5f1c7`) is fixed at `ff1b659`. Exact count: 0 FAIL rows; `BDefrostController` ABSENT from output.

The delay-floor rule remains pinned by LD1 (`Math.max(x, 0L)` → FAIL), LD3 (`BRelTime.make(0)` → FAIL), LD6 (`schedulePeriodically` zero-floor → FAIL) on synthetic fixtures — no new fixture is owed.

`[ev: R-T2.6]` `[ev: R-T2.7]` `[ev: proposal SC-7]` `[ev: memory coldroompan-defrost-time-le-0-bug]`

---

### RC8-unchanged (RC8 still 1 FAIL at ff1b659)

**Given** `DashboardPan-ux` at `Leon-Guanjuato-worktrees/main-ff1b659` (`ff1b659`). `[ev: K21]`

**When** `rc-scan.sh` runs (via the exported variable from `client-root.bash`).

**Then** exits **1** with a `FAIL` row naming `host` (the `:701` host literal in `index.html`). Exact count: 1 FAIL row; subject: `host` (or the specific host literal); no new FAIL rows introduced. The verdict is identical on both `4f5f1c7` and `ff1b659`.

`[ev: R-T2.9]` `[ev: proposal SC-7]` `[ev: lead re-measure 2026-09-07]`

---

## Success criteria (this capability)

- [ ] `tests/lib/client-root.bash` exists, exports `C9_CLIENT_ROOT`, `C9_CLIENT_REPO`, `C8_CLIENT_REPO` with the `main-ff1b659` default.
- [ ] Absolute `Leon-Guanjuato` literals in `tests/*.bats` outside the lib: **10 → 0** (OBSERVED).
- [ ] Suite green with all three variables **unset** from the environment.
- [ ] One override pin per variable showing env wins over the default.
- [ ] LD5: exits **0**, **0 FAIL rows**, `BDefrostController` ABSENT (retargeted to `ff1b659` — was exit 1 + FAIL on `4f5f1c7`).
- [ ] c8-close SC1-smoke: re-pinned to `ff1b659` clean state.
- [ ] RC8: **1 FAIL row**, subject `host`, exit 1 — **UNCHANGED** from `dab0807` behavior.
- [ ] No real-tree smoke in the suite asserts FAIL without a named ticket or an existing synthetic rule carrier (R-T2.10).
- [ ] No toolbelt script is touched by PR3.
- [ ] `shellcheck 0.10.0` exits 0; 0 attribution trailers (K11).
