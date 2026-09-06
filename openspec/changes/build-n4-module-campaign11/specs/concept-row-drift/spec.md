# Spec: concept-row-drift advisory (T3)

**Capability**: `module-write-path-matrix` — `lint-write-path.sh` DRIFT class
**PR**: PR2 (`feat/c11-concept-row-drift`)
**QA RED**: `qa/c11-concept-drift` **`77352a7`** (base `dab0807`; WP-drift-neg / WP-drift-strict RED on dab0807, WP-drift-true-concept + WP-drift-decoy guards; decoy verified to flip if the comment strip is dropped) — the RED is the contract (K13). `[ev: proposal PR2 row; explore §3.2]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K19/K20/K21/K22/K24(7)/D9b/comment-strip/observed-flip/real-tree-smoke/fragment-merge)
**Seam**: `lint-write-path.sh` S25 row pass `:422-458` — specifically the `[concept]` skip at `:441` (`case "$_row" in *'[concept]'*) continue`).

---

## Context — STALE (S25, C10) vs DRIFT (T3, C11)

STALE (landed in C10, `v0.21.0`): a matrix row whose backtick-inner slot name is NOT in the covered set AND does NOT carry `[concept]` → STALE advisory.
DRIFT (NEW in C11): the **inverse** — a `[concept]`-marked matrix row whose backtick-inner slot name **IS** now in the covered set → DRIFT advisory. A true concept row (slot name absent from source) stays silent.

STALE and DRIFT are complementary guards in the same lint; both live in the S25 row pass.

---

## Delta — ADDED requirements

The following ADD the DRIFT advisory class to `lint-write-path.sh`. The STALE class, the uncovered FAIL, and exit 3 are UNCHANGED — byte-identical to `v0.21.0`.

### DRIFT detection

| ID | Requirement | ev |
|----|-------------|-----|
| R-T3.1 | `lint-write-path.sh` MUST detect **DRIFT rows**: a matrix row that carries the literal token `[concept]` anywhere in the row AND whose backtick-inner slot name IS in the covered set (all `@NiagaraProperty` names (any flag) ∪ all `@NiagaraAction` names ∪ `--bog` extras, harvested matrix-root-wide) is a DRIFT row — its `[concept]` marker is stale. | `[ev: apply-package 4ef4f864c §T3]` `[ev: lint-write-path.sh:441 @ dab0807 — the skipped branch that becomes the test]` |
| R-T3.2 | A `[concept]` token that appears **only inside a markdown comment** (`<!-- … -->`) or inside a `//` single-line comment is stripped before matching (cross-cutting comment-strip rule) and MUST NOT exempt the row or trigger DRIFT. | `[ev: ../cross-cutting.md — comment-strip rule]` `[ev: proposal SC-5 — comment decoy]` |
| R-T3.3 | DRIFT detection is **per-row**: one DRIFT advisory row per matching matrix row. A row whose `[concept]` marker is stale does not affect other rows with the same or a different slot name. | `[ev: apply-package 4ef4f864c §T3]` |
| R-T3.4 | The **covered set** is the same matrix-root-wide union used by the STALE check (all `@NiagaraProperty` names (any flags), all `@NiagaraAction` names, `--bog` extras). The multi-line `name = "X"` field-line harvest MUST be used (K24(5), which fixed the 56 vs 177 undercount at `ff1b659`). The seam is the existing matrix-root-wide harvest at `lint-write-path.sh:144-149`. | `[ev: K24(3)]` `[ev: K24(5) — multi-line name field fix]` `[ev: lint-write-path.sh:144-149 @ dab0807]` |
| R-T3.5 | DRIFT row grammar (STATUS-first, same shape as the STALE row): `DRIFT  lint-write-path  <matrix-path>:<line>  slot <name>: concept marker but a source slot exists`. | `[ev: apply-package 4ef4f864c §T3]` `[ev: lint-write-path.sh:374 @ dab0807 — shape reference]` |

### Exit-code contract (DRIFT added, others unchanged)

| ID | Requirement | ev |
|----|-------------|-----|
| R-T3.6 | Without `--strict`: DRIFT rows are printed to stdout as advisory rows; exit **0**. | `[ev: proposal R2]` `[ev: K20]` |
| R-T3.7 | Under `--strict`: DRIFT rows promote to exit **1** — same flag as STALE. | `[ev: proposal R2]` `[ev: K20]` |
| R-T3.8 | The uncovered **FAIL** (a source OPERATOR slot with no matrix row) exits **1** both with and WITHOUT `--strict`. This is **byte-identical to `v0.21.0`** — not weakened. | `[ev: proposal SC-5]` `[ev: K20]` `[ev: lint-write-path.sh:374,383 @ dab0807]` |
| R-T3.9 | Exit **3** on usage/no-matrix is **UNCHANGED**. | `[ev: K20]` `[ev: lint-write-path.sh:383 @ dab0807]` |

### True concept row — SILENT

| ID | Requirement | ev |
|----|-------------|-----|
| R-T3.10 | A `[concept]` row whose backtick-inner slot name is **NOT** in the covered set (a genuine concept marker — the concept has not yet been implemented as a source slot) MUST produce **no DRIFT row** and MUST NOT be treated as STALE. Such a row remains silently exempt. | `[ev: proposal R2 — inverse of STALE; true concept stays silent]` `[ev: apply-package 4ef4f864c §T3]` |

### Real-tree pin

| ID | Requirement | ev |
|----|-------------|-----|
| R-T3.11 | `lint-write-path.sh` on client tree `00e7118` (= `ff1b659` + PR6, which added the five `[concept]` rows for `hoaMode`, `inhibit`, `freezeEnabled`, `setpoint`, `coolOnSensorFault`) MUST report **0 DRIFT rows** (default exit 0). The five PR6 concept rows mark slots that have no source `@NiagaraProperty` or `@NiagaraAction` at `00e7118`; they are TRUE concept rows, not stale ones. | `[ev: proposal SC-5]` `[ev: explore §2 client state — 00e7118 = ff1b659 + PR6 five concept rows]` |

---

## Scenarios

### T3-drift-pos (DRIFT row printed, default exit 0)

**Given** a `docs/write-path-matrix.md` with one row containing `[concept]` and whose backtick-inner slot name (`setpoint`) IS declared as `@NiagaraProperty` in a Java source under the matrix root. The `[concept]` token appears in the row text (not inside a markdown comment). A comment-only decoy row contains `<!-- [concept] setpoint -->` which MUST NOT trigger DRIFT.

**When** `lint-write-path.sh` runs (default, no `--strict`).

**Then** exits **0** with exactly one `DRIFT  lint-write-path  <matrix-path>:<line>  slot setpoint: concept marker but a source slot exists` row. The comment-decoy row produces **no DRIFT row**.

`[ev: R-T3.1]` `[ev: proposal SC-5 DRIFT pos + decoy]`

---

### T3-drift-strict (exit 1 under --strict)

**Given** the same DRIFT matrix as T3-drift-pos.

**When** `lint-write-path.sh --strict` runs.

**Then** exits **1**.

`[ev: R-T3.7]` `[ev: proposal SC-5]`

---

### T3-concept-silent (true concept row stays silent)

**Given** a matrix with a row containing `[concept]` and whose slot name (`airDefrost`) is NOT in any `@NiagaraProperty`, `@NiagaraAction`, or `--bog` entry in the source tree.

**When** `lint-write-path.sh` runs (default and with `--strict`).

**Then** exits **0** with **0 DRIFT rows** and **0 STALE rows** for `airDefrost`. The concept marker correctly exempts the row because the concept is not yet implemented.

`[ev: R-T3.10]` `[ev: proposal SC-5 — true concept row silent]`

---

### T3-comment-decoy (concept in HTML comment does not exempt or trigger)

**Given** a matrix with a row that does NOT carry `[concept]` in its text content, but a preceding markdown comment block `<!-- [concept] setpoint -->` exists in the file. The row's slot name (`setpoint`) IS in the covered set.

**When** `lint-write-path.sh` runs.

**Then** the row is NOT exempt from STALE detection (if it has no match in source — it IS a STALE row). The comment's `[concept]` token is stripped before matching and MUST NOT exempt or trigger DRIFT.

`[ev: R-T3.2]` `[ev: ../cross-cutting.md — comment-strip rule]`

---

### T3-stale-unchanged (STALE class byte-identical to v0.21.0)

**Given** a matrix with one row whose slot name is NOT in the covered set and does NOT carry `[concept]` (a STALE row, as in the S25 spec).

**When** `lint-write-path.sh` runs with and without `--strict`.

**Then** the STALE row grammar, count, and exit codes are **identical** to the behavior at `v0.21.0` (`dab0807`). No STALE row is changed to DRIFT.

`[ev: R-T3.8]` `[ev: S25 spec @ C10 archive]`

---

### T3-uncovered-fail-unchanged (uncovered FAIL unchanged)

**Given** a source OPERATOR slot `freezeEnabled` with no matrix row at all.

**When** `lint-write-path.sh` runs with and without `--strict`.

**Then** exits **1** in both cases with a `FAIL` row naming `freezeEnabled`. The DRIFT addition does NOT weaken the uncovered FAIL gate.

`[ev: R-T3.8]` `[ev: K20]`

---

### T3-real-tree-00e7118 (real-tree pin — 0 DRIFT at client 00e7118)

**Given** any module root that resolves the `docs/write-path-matrix.md` at client `00e7118` (`Leon-Guanjuato-worktrees/main-ff1b659` + the five PR6 concept rows). `[ev: K21 — worktree cite + commit]`

**When** `lint-write-path.sh` runs.

**Then** exits **0** with **0 DRIFT rows**. The five `[concept]` rows introduced by PR6 (`hoaMode`, `inhibit`, `freezeEnabled`, `setpoint`, `coolOnSensorFault`) name slots that have no source declaration at `00e7118`; they are true concept rows and MUST remain silent.

`[ev: R-T3.11]` `[ev: proposal SC-5]`

---

### T3-observed-flip (OBSERVED mutation — DRIFT disappears when slot is removed from source)

**Given** the T3-drift-pos fixture where `setpoint` IS in the source.

**When** the `setpoint` `@NiagaraProperty` declaration is removed from the fixture source:

**Then** `lint-write-path.sh` exits **0** with **0 DRIFT rows** for `setpoint` — the concept marker is now correct again (OBSERVED flip: DRIFT present → DRIFT absent). This mutation confirms the covered-set test is necessary.

`[ev: K24(7)]` `[ev: ../cross-cutting.md — observed-flip]`

---

## Success criteria (this capability)

- [ ] T3-drift-pos: exits 0 with exactly 1 DRIFT row; T3-drift-strict: exits 1.
- [ ] T3-concept-silent: true concept row (slot absent) → 0 DRIFT and 0 STALE, exit 0.
- [ ] T3-comment-decoy: `[concept]` in HTML comment does NOT exempt or trigger DRIFT.
- [ ] STALE class behavior: byte-identical to `v0.21.0` with and without `--strict`.
- [ ] Uncovered FAIL exit 1: identical with and without `--strict`.
- [ ] Exit 3 preserved (K20 disjoint).
- [ ] Real tree at client `00e7118`: **0 DRIFT rows**, exit 0.
- [ ] OBSERVED flip: removing the source slot makes the DRIFT row disappear (naming the fixture).
- [ ] Every new fixture strips `//`/`/* */` before matching and carries a comment-only decoy.
- [ ] `kit-links.bats` green; `--strict` flag row for DRIFT added in `BUILD-LOOP.md` and `skill/SKILL.md` (K19); `shellcheck 0.10.0` exits 0; 0 attribution trailers (K11).
