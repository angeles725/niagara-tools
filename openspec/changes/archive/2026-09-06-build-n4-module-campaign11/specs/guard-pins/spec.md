# Spec: guard-pins meta-check (T4)

**Capability**: `module-guard-pin-lint` — `toolbelt/lint-guard-pins.sh`
**PR**: PR4 (`feat/c11-lint-guard-pins`)
**QA RED**: `qa/c11-guard-pins` (being authored on `dab0807`) — PR does not open until RED is blessed. `[ev: explore §3.2; proposal PR4 row]`
**Repo**: `niagara-tools` (kit)
**Cross-cutting**: see `../cross-cutting.md` (K11/K12/K13/K19/K20/K21/K22/K24(7)/D9b/comment-strip/observed-flip/fragment-merge)
**Rationale**: K24(7) is doctrine (every mutation must name the fixture it flips, QA confirms) but unenforced: C10 caught three unpinned guards by hand. C11 automates that check. `[ev: 8ad4bb36e §T4]` `[ev: K24(7)]`

---

## Delta — NEW requirements

### What `lint-guard-pins.sh` checks

| ID | Requirement | ev |
|----|-------------|-----|
| R-T4.1 | `toolbelt/lint-guard-pins.sh` MUST scan every lint file under `toolbelt/` (pruning dot-dirs, D9b) for header lines that declare a mutation. For each declared mutation name, the lint MUST verify that at least one bats fixture whose name matches the declared mutation name exists in `tests/*.bats` (or a subdirectory visible from there). A declared mutation with no matching fixture MUST produce a WARN row. | `[ev: 8ad4bb36e §T4]` `[ev: K24(7)]` |
| R-T4.2 | **No mutation line = WARN**. A lint header that declares ZERO mutation lines MUST itself produce a WARN row — the lint cannot pass vacuously by omitting all `# Mutation:` / `# OBSERVED mutation:` declarations. This ensures the guard cannot be bypassed by leaving the header blank. | `[ev: proposal §T4 — "so the lint cannot be vacuously clean"]` `[ev: 8ad4bb36e §T4]` |
| R-T4.3 | The exact syntax of the mutation declaration header line (the grammar for `# Mutation:` / `# OBSERVED mutation:` and how fixture names are extracted from it) is defined in `sdd-design`. The spec requires only that: (a) each lint header has at least one declared mutation (R-T4.2), and (b) each declared mutation name maps to an existing bats fixture (R-T4.1). Design owns the exact grammar. | `[ev: proposal RK7 — "T4 header grammar is undefined — a loose parser yields 0 WARN vacuously"]` `[ev: 8ad4bb36e §T4]` |
| R-T4.4 | WARN row grammar (STATUS-first): `WARN  lint-guard-pins  <lint-file>:<line>  <description>` where `<description>` distinguishes "missing fixture for declared mutation X" from "no mutation line declared". | `[ev: 8ad4bb36e §T4]` `[ev: lint-write-path.sh:374 @ dab0807 — STATUS-first shape reference]` |

### Exit-code contract (K20 disjoint)

| ID | Requirement | ev |
|----|-------------|-----|
| R-T4.5 | Exit **0** (WARN-only): no mutation problems found, or problems found but `--strict` not specified. | `[ev: proposal R4]` `[ev: K20]` |
| R-T4.6 | Exit **1** under `--strict`: any WARN promoted to exit 1. | `[ev: proposal R4]` `[ev: K20]` |
| R-T4.7 | Exit **3** on usage or missing-toolkit-root: no argument provided, or the toolbelt directory does not exist. | `[ev: proposal R4]` `[ev: K20]` |

### D9b — dot-dir prune

| ID | Requirement | ev |
|----|-------------|-----|
| R-T4.8 | `lint-guard-pins.sh` MUST prune dot-directories (`.git`, `.cache`, etc.) when traversing the `toolbelt/` directory. A mutation declared in a file inside a dot-directory MUST NOT be scanned. | `[ev: D9b]` `[ev: ../cross-cutting.md — D9b]` |

### Self-verification gate (run over the kit)

| ID | Requirement | ev |
|----|-------------|-----|
| R-T4.9 | When `lint-guard-pins.sh` is run over the kit at `dab0807` PLUS the changes introduced by PR1, PR2, and PR3, it MUST report **0 WARN rows** (exit 0). If it finds any WARN rows, those are the finding, and they MUST be fixed in the same PR4 before merge. The lead gate includes this self-verification run as a mandatory step. | `[ev: proposal PR4 row — "run over the kit at dab0807 + PR1..PR3 → expected 0 WARN"]` `[ev: K24(7)]` |

---

## Scenarios

### T4-pos (WARN on missing fixture — RED: must WARN)

**Given** a synthetic lint file in `toolbelt/` whose header declares `# Mutation: SomeMutation` but no bats fixture named `SomeMutation` (or containing `SomeMutation`) exists in `tests/*.bats`.

**When** `lint-guard-pins.sh` runs over the `toolbelt/` directory.

**Then** exits **0** with exactly **1 WARN row** naming the lint file and `SomeMutation`. On `dab0807` this fixture exits 0 with no WARN (the tool does not exist — C11-T4-pos is RED — OBSERVED).

`[ev: R-T4.1]` `[ev: proposal PR4 row — positive fixture: 1 WARN]`

---

### T4-no-mutation-line (WARN on missing mutation declaration — RED: must WARN)

**Given** a synthetic lint file in `toolbelt/` whose header contains no `# Mutation:` or `# OBSERVED mutation:` line at all.

**When** `lint-guard-pins.sh` runs.

**Then** exits **0** with exactly **1 WARN row** for the lint file, indicating it has no declared mutation (R-T4.2).

`[ev: R-T4.2]` `[ev: proposal T4 — "a header with no named mutation is itself a WARN"]`

---

### T4-neg (clean — every named mutation has a fixture)

**Given** all `toolbelt/*.sh` files whose headers declare mutations, where every declared mutation name corresponds to at least one bats fixture in `tests/*.bats`.

**When** `lint-guard-pins.sh` runs.

**Then** exits **0** with **0 WARN rows**.

`[ev: R-T4.1]` `[ev: proposal SC-8 — negative: 0 WARN]`

---

### T4-strict (exit 1 under --strict)

**Given** the T4-pos fixture (1 WARN row present).

**When** `lint-guard-pins.sh --strict` runs.

**Then** exits **1**.

`[ev: R-T4.6]` `[ev: K20]`

---

### T4-usage (exit 3)

**Given** `lint-guard-pins.sh` is invoked with no arguments.

**When** it runs.

**Then** exits **3**.

`[ev: R-T4.7]` `[ev: K20]`

---

### T4-dotdir-prune (D9b)

**Given** a `.hidden/` directory under `toolbelt/` containing a lint file that declares a mutation with no corresponding fixture.

**When** `lint-guard-pins.sh` runs.

**Then** exits **0** with **0 WARN rows** (the dot-directory was pruned and the hidden file was not scanned).

`[ev: R-T4.8]` `[ev: D9b]`

---

### T4-self-verify (0 WARN over kit at dab0807 + PR1..PR3)

**Given** the kit at `dab0807` with the changes from PR1 (adds `toolbelt/lib/method-boundary.sh`), PR2, and PR3 applied.

**When** `lint-guard-pins.sh` runs over the `toolbelt/` directory.

**Then** exits **0** with **0 WARN rows**. If any WARN rows are printed, they are the finding and MUST be closed in PR4 before merge.

`[ev: R-T4.9]` `[ev: proposal SC-8]`

---

### T4-observed-flip (OBSERVED mutation — WARN appears when fixture is deleted)

**Given** the T4-neg state (every mutation has a fixture).

**When** a bats fixture that is the sole carrier for a named mutation is deleted.

**Then** `lint-guard-pins.sh` exits **0** with **1 WARN row** for the now-unmatched mutation (OBSERVED flip: clean → 1 WARN). This confirms the fixture-existence check is load-bearing.

`[ev: K24(7)]` `[ev: ../cross-cutting.md — observed-flip]`

---

## Success criteria (this capability)

- [ ] T4-pos: exits 0 with exactly 1 WARN row for a missing fixture (RED on `dab0807` — OBSERVED).
- [ ] T4-no-mutation-line: exits 0 with exactly 1 WARN row when no mutation line is declared.
- [ ] T4-neg: exits 0 with 0 WARN rows when all declared mutations have fixtures.
- [ ] T4-strict: exits 1 when 1 WARN is present under `--strict`.
- [ ] T4-usage: exits 3 with no arguments.
- [ ] T4-dotdir-prune: `.hidden/` lint file not scanned (D9b).
- [ ] T4-self-verify: run over kit at `dab0807` + PR1..PR3 → **0 WARN rows**, or the list IS the finding and is closed in PR4.
- [ ] OBSERVED flip: removing a bats fixture for a named mutation produces 1 WARN (confirming the check is load-bearing).
- [ ] `kit-links.bats` green with `lint-guard-pins.sh` routed in both `BUILD-LOOP.md` and `skill/SKILL.md` (K19).
- [ ] Exit codes disjoint: 0/1/3 (K20); dot-dirs pruned (D9b); `shellcheck 0.10.0` exits 0; 0 attribution trailers (K11).
