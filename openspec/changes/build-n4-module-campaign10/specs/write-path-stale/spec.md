# Spec: write-path STALE advisory (S25)

**Capability**: `module-write-path-matrix` — `lint-write-path.sh` STALE class + `--strict`
**PR**: PR5 (`feat/c10-write-path-stale`)
**QA RED**: `qa/c10-write-path-strict` **`db130a7`** (base `df8c7ec`, final S25 contract; `a56a72e` + header grammar align) — `lint-write-path.bats` pins WP-stale-neg, -strict, -regression, -concept, -concept-decoy, -perrow, -prose, -action, -summary, -smoke, plus WP-uncovered-strict; WP1/WP2 unchanged. The RED is the contract (K13).
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K14/K19/K20/D9b/comment-strip/observed-flip/real-tree-smoke)
**Seams** (per companero `fb3bece`): `:161` matrix-slot awk, `:310` scanner, `:374` FAIL emit, `:383` exit.

---

## Delta — ADDED/MODIFIED requirements

The following ADD the STALE advisory class and `--strict` flag to `lint-write-path.sh`. The uncovered FAIL (a source OPERATOR slot with no matrix row) is UNCHANGED — byte-identical to v0.20.0.

### STALE detection

| ID | Requirement | ev |
|----|-------------|-----|
| R-S25.1 | `lint-write-path.sh` MUST detect **STALE rows**: a row in `docs/write-path-matrix.md` whose backtick-inner slot name is NOT in the set (`all @NiagaraProperty names (any flag) ∪ all @NiagaraAction names ∪ --bog extras`) AND which does NOT itself carry the literal token `[concept]` is a STALE row. | `[ev: proposal R5]` `[ev: seam :161 matrix-slot awk fb3bece]` |
| R-S25.2 | STALE is **per-row** — one STALE advisory row per matching matrix row. A `[concept]` marker exempts only its own row; it does NOT exempt other rows with the same slot name. | `[ev: coordinator 2026-09-07]` |
| R-S25.3a | The covered set is harvested from EVERY Java source under the **matrix root** (the directory holding `docs/write-path-matrix.md`, which the lint already walks up to), across all modules, with `build/` and dot-dirs pruned (D9b) — never from the invoked module root alone. The STALE count is therefore identical whichever module root is passed. The uncovered FAIL stays per-module as today. | `[ev: QA 2026-09-07 — a per-module covered set flags every other module's real slots]` |
| R-S25.3 | The **covered set** includes ALL `@NiagaraProperty` names (any flags, including SUMMARY-only and non-OPERATOR), ALL `@NiagaraAction` names, and `--bog` extras — NOT the OPERATOR-only `:310`/`:338` scanner output. A matrix row naming an `@NiagaraAction` (e.g. `intervalExpired`, `forceDefrost`) is therefore NOT STALE. A matrix row naming a SUMMARY-only (non-OPERATOR) property is NOT STALE. | `[ev: coordinator 2026-09-07]` `[ev: docs/write-path-matrix.md :64,:65 @ ff1b659 action rows]` |
| R-S25.4 | STALE row grammar (STATUS-first, same shape as the existing FAIL row at `lint-write-path.sh:374`): `STALE  lint-write-path  <matrix-path>:<line>  slot <name>: no source slot with that name`. The subject carries the matrix line so the three `hoaMode` rows are distinguishable. | `[ev: design D5b]` `[ev: lint-write-path.sh:374 @ cb79676]` |
| R-S25.5 | `[concept]` exemption: a matrix row that carries the **literal** token `[concept]` anywhere in the row is NOT STALE, regardless of whether the slot name appears in source. A `[concept]` token inside a markdown comment (`<!-- … -->` or `//`) is stripped before matching and does NOT exempt the row. | `[ev: coordinator 2026-09-07]` `[ev: cross-cutting.md — comment-strip rule]` |

### `--strict` flag

| ID | Requirement | ev |
|----|-------------|-----|
| R-S25.6 | Without `--strict`: STALE rows are printed to stdout as advisory rows; exit **0** (advisory only). | `[ev: proposal SC-5]` `[ev: seam :383 fb3bece]` |
| R-S25.7 | Under `--strict`: STALE rows promote to exit **1**. | `[ev: proposal SC-5]` `[ev: K20]` |

### Uncovered FAIL — UNCHANGED

| ID | Requirement | ev |
|----|-------------|-----|
| R-S25.8 | The existing uncovered FAIL (a source OPERATOR slot with no matrix row) exits **1** both with and WITHOUT `--strict`. This gate is **byte-identical to v0.20.0** — not weakened. | `[ev: proposal SC-5]` `[ev: K20]` |
| R-S25.9 | Exit **3** on usage/no-matrix is **unchanged**. | `[ev: K20]` `[ev: lint-write-path.sh:383 @ cb79676]` |

### Real-tree pin (DashboardPan at ff1b659 — BEFORE S26 edit)

| ID | Requirement | ev |
|----|-------------|-----|
| R-S25.10 | `lint-write-path.sh` on DashboardPan at client `ff1b659` **BEFORE** the S26 client edit MUST report **exactly 5 STALE rows** (default exit 0): matrix rows at `:31`, `:32`, `:52` (all named `hoaMode`), `:33` (named `inhibit`), `:36` (named `freezeEnabled`). All five lack a matching `@NiagaraProperty`, `@NiagaraAction`, or `--bog` entry (`inhibit` verified NOT bog-traced, read-only on the PANCCADIA config.bog); only backtick-inner `^[a-z][A-Za-z0-9]*$` cells count (`:40` is `setpoint`, covered). The same 5 MUST be reported from a second module root (CompPan-rt) — pins the matrix-root scope. | `[ev: companero a012de135 + QA a56a72e WP-stale-smoke]` |
| R-S25.11 | After the S26 client edit marks those rows with `[concept]`, `lint-write-path.sh` on DashboardPan at `ff1b659` MUST report **0 STALE rows** (the OBSERVED real-tree flip). | `[ev: coordinator 2026-09-07]` |

---

## Scenarios

### WP-stale-neg (must-pass — STALE row printed, default exit 0)

**Given** a matrix with one row whose backtick-inner slot name (`hoaMode`) appears in no `@NiagaraProperty`, `@NiagaraAction`, or `--bog` entry, and that row does NOT carry `[concept]`.

**When** `lint-write-path.sh` runs (default, no `--strict`).

**Then** exits **0** with exactly one `write-path  STALE  hoaMode  no source slot with that name` row printed.

`[ev: proposal R5 WP-stale-neg]`

---

### WP-stale-strict (exit 1 under --strict)

**Given** the same STALE matrix as WP-stale-neg.

**When** `lint-write-path.sh --strict` runs.

**Then** exits **1**.

`[ev: proposal R5 WP-stale-strict]`

---

### WP-stale-concept-exempt (concept marker exempts the marked row only)

**Given** a matrix with two rows for the same slot name `hoaMode`: one carrying `[concept]` and one without.

**When** `lint-write-path.sh` runs.

**Then** exits **0** with exactly **1 STALE row** for the unmarked `hoaMode` row. The concept-marked row produces **no STALE row**.

`[ev: coordinator 2026-09-07 — a marked row never exempts another row with the same name]`

---

### WP-stale-concept-exit-strict (concept row does not trigger exit 1 under --strict)

**Given** a matrix where the ONLY non-covered row carries `[concept]`.

**When** `lint-write-path.sh --strict` runs.

**Then** exits **0** (the `[concept]` row is NOT STALE; --strict has nothing to promote).

`[ev: coordinator 2026-09-07]`

---

### WP-action-row-covered (action names are covered)

**Given** a matrix with a row naming an `@NiagaraAction` (`intervalExpired`).

**When** `lint-write-path.sh` runs against a source tree that declares `@NiagaraAction(name="intervalExpired")`.

**Then** **no STALE row** for `intervalExpired`.

`[ev: coordinator 2026-09-07]` `[ev: docs/write-path-matrix.md :64 @ ff1b659]`

---

### WP-summary-row-covered (SUMMARY-only property names are covered)

**Given** a matrix row naming a property that is declared with `@NiagaraProperty` but with only `SUMMARY` flags (no `OPERATOR`).

**When** `lint-write-path.sh` runs.

**Then** **no STALE row** for that property (covered by the union of all `@NiagaraProperty` names).

`[ev: coordinator 2026-09-07]`

---

### WP-stale-regression (matching matrix — no STALE row)

**Given** a matrix where every row's slot name matches a source `@NiagaraProperty` or `@NiagaraAction` name.

**When** `lint-write-path.sh` runs.

**Then** exits **0** with **no STALE row**.

`[ev: proposal R5 WP-stale-regression]`

---

### WP-uncovered-fail-default (uncovered FAIL unchanged without --strict)

**Given** a source OPERATOR slot `setpoint` with **no matrix row**.

**When** `lint-write-path.sh` runs WITHOUT `--strict`.

**Then** exits **1** with `FAIL  lint-write-path  <module>  slot setpoint: no matrix row`.

`[ev: proposal SC-5]` `[ev: R-S25.8]`

---

### WP-uncovered-fail-strict (uncovered FAIL unchanged with --strict)

**Given** the same uncovered OPERATOR slot.

**When** `lint-write-path.sh --strict` runs.

**Then** exits **1** — same as without `--strict` (FAIL gate is unchanged).

`[ev: K20]` `[ev: R-S25.8]`

---

### WP-real-tree-dashboard (real-tree before S26)

**Given** DashboardPan module root at client `ff1b659` (worktree `Leon-Guanjuato-worktrees/main-ff1b659`), BEFORE any S26 client edit.

**When** `lint-write-path.sh` runs (default, no `--strict`).

**Then** exits **0** with exactly **5 STALE rows**: slot names `hoaMode` (3 rows), `inhibit` (1 row), `freezeEnabled` (1 row) — all five printed, no FAIL rows.

`[ev: R-S25.10]` `[ev: coordinator 2026-09-07]`

---

### WP-real-tree-dashboard-post-s26 (real-tree after S26)

**Given** DashboardPan module root at client `ff1b659`, AFTER the S26 client edit adds `[concept]` to those five rows.

**When** `lint-write-path.sh` runs.

**Then** exits **0** with **0 STALE rows** (OBSERVED real-tree flip from 5 → 0).

`[ev: R-S25.11]` `[ev: coordinator 2026-09-07]`

---

## Success criteria (this capability)

- [ ] WP-stale-neg: exits 0 with 1 STALE row; WP-stale-strict: exits 1.
- [ ] WP-stale-concept-exempt: 2-rows-same-name, 1 marked → exactly 1 STALE row.
- [ ] WP-action-row-covered: `intervalExpired` not STALE; WP-summary-row-covered: SUMMARY-only not STALE.
- [ ] WP-stale-regression: matching matrix → 0 STALE rows, exit 0.
- [ ] Uncovered FAIL exit 1 is BYTE-IDENTICAL with and without `--strict`.
- [ ] Exit 3 preserved (K20 disjoint).
- [ ] DashboardPan at `ff1b659` before S26: **5 STALE rows** (hoaMode ×3, inhibit, freezeEnabled); after S26: **0 STALE rows** (OBSERVED flip).
- [ ] `kit-links.bats` green; `--strict` flag row added in SKILL.md and BUILD-LOOP.md (K19); `shellcheck 0.10.0` exits 0; 0 attribution trailers.
