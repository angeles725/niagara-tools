# Modular modules by default (2026-10-27)

Locator: `build-n4-module-kit/odd/tasks/modularity-2026-10-27.md` · branch `docs/fold-r10-retros` (on top of the r10
retro fold, tip db22d23e; base origin/main 6c34e20c) · worktree `niagara-tools-worktrees/fold-r10-retros`.

## Objective
A module built with the kit is modular by default: every module root carries a `MODULE-MAP.md` that names each
concern's files and tests, a lint checks the map against the tree, the scaffold writes a strict map for every type,
and the agent rule says to read the map before editing. Existing (legacy) modules keep warnings; new (strict)
modules get failing rows.

## Problem / why
A read-only audit of four client modules (A to D, anonymized; this repo is public) found single adapter classes of
700 to 1400 hand-written lines, pure cores with 220-line methods, a 3260-line single-file dashboard front end and
servlets with 28 handlers. The existing gates did not stop it: L3 passes a module with no `model/` package, the
lint-size method rule skips B-classes and wrapped signatures, the slot wall is a WARN, and nothing tells an agent
which file owns a concern. Evidence and host file:line pointers are in the three retros below.

## Scope and constraints
- Part 1 (this worker): M1 to M3. Part 2 (a later worker): M4 to M6.
- Allowed edit surfaces are the ones the parent passed (kit docs, the three retros, scaffold and fixtures,
  `go/internal/lint` paths and the new `rules/modulemap`, reportmodule, lintindex, `tests/`, `changelog.d/`).
- `retros/INDEX.md`, `toolbelt/INDEX.md` and `BUILD-STATE.md` are generated and not committed.
- The retros stay `review-status: pending` until part 2 finishes; this document cites deltas in plain text only
  (never as evidence tags), so it credits nothing in the fold audit by itself.
- Public repo: no client names, paths, hosts or IPs; client modules are "client module A/B/C/D" with counts only.

## Settled parent decisions
1. A missing map is a WARN only (MM1); the BUILD-STATE mirror surfaces it at orient. Lints stay kit-repo-free.
2. MM4: Java files are listed one per entry; front-end `src/rc` entries may be directory globs.
3. The servlet delta (gaps Δ6) lands after the r10 fold (already done on this branch).
4. The fixture move (strict-new Δ2) lands with strict L3 (strict-new Δ3).
5. The adapter decision budget (gaps Δ5) stays WARN until calibrated.

## Delta mapping (proposal number → retro and Δ within it)
| Proposal | Retro (stem) | Δ in retro | Task |
|---|---|---|---|
| Δ1 | modularity-module-map | Δ1 | M1 |
| Δ2 | modularity-module-map | Δ2 | M3 |
| Δ3 | modularity-module-map | Δ3 | M2 |
| Δ4 | modularity-module-map | Δ4 | M1 |
| Δ18 | modularity-module-map | Δ5 | M1 |
| Δ5 | modularity-strict-new | Δ1 | M2 |
| Δ6 | modularity-strict-new | Δ2 | M3 |
| Δ7 | modularity-strict-new | Δ3 | M4 |
| Δ8 | modularity-strict-new | Δ4 | M4 |
| Δ9 | modularity-strict-new | Δ5 | M4 |
| Δ10 | modularity-gaps | Δ1 | M4 |
| Δ11 | modularity-gaps | Δ2 | M5 |
| Δ12 | modularity-gaps | Δ3 | M5 |
| Δ13 | modularity-gaps | Δ4 | M5 |
| Δ14 | modularity-gaps | Δ5 | M4 |
| Δ15 | modularity-gaps | Δ6 | M5 |
| Δ16 | modularity-gaps | Δ7 | M1 |
| Δ17 | modularity-gaps | Δ8 | M5 |

## Tasks
- [x] M0 · open the campaign: this document and the three pending retros (WU0). Route: delegated writer.
- [ ] M1 · docs (WU1): module-map Δ1 (MODULE-MAP schema § in `types/structure.md`, METHODOLOGY conformance item),
  module-map Δ4 (agent rule in `skill/SKILL.md`), gaps Δ7 (test naming in `types/moduleTest.md`), module-map Δ5
  (`modularity` field in `BUILD-STATE.src.md`). Route: delegated writer. Checks: strict fold audit; kit-links and
  doc-pin bats.
- [ ] M2 · Go, test-first (WU2): strict-new Δ1 `lint.ModuleMode(root)` with table tests; module-map Δ3 rule
  package `rules/modulemap` (MM1 to MM6), registration, wrapper `toolbelt/lint-module-map.sh`, goldens. Route:
  delegated writer. Checks: `go test`/`go vet` on `./internal/lint/...`; bats `tests/lint-module-map.bats`;
  `gen-lint-index.sh --check`; shellcheck.
- [ ] M3 · scaffold, test-first (WU3): module-map Δ2 (each fixture carries a strict `MODULE-MAP.md`, so the scaffold
  emits one per type); strict-new Δ2 (fixture pure classes into `model/`); bats asserting a fresh scaffold of every
  type passes `lint-module-map` with 0 FAIL. Route: delegated writer. Checks: `tests/scaffold-module.bats`;
  shellcheck.
- [ ] M4 · strict rules (part 2): strict-new Δ3 (strict L3), Δ4 (strict lint-size), Δ5 (slot wall); gaps Δ1
  (model import direction), Δ5 (adapter decision budget, WARN).
- [ ] M5 · front end, servlet, report (part 2): gaps Δ2 and Δ3 (rc-scan), Δ4 (dashboard JS skeleton), Δ6 (servlet
  router + handlers), Δ8 (report-module modularity summary).
- [ ] M6 · close (part 2): flip the three retros to folded, strict fold audit 0 uncited, ledger fragment, CI-local.

## Acceptance criteria
- `lint-module-map` reports MM1 to MM6 as specified, is registered in `rules.All()` and has a toolbelt wrapper
  whose header passes `gen-lint-index.sh --check`.
- A fresh scaffold of logic, dashboard and driver passes `lint-module-map` with 0 FAIL.
- The schema, agent rule, test naming and BUILD-STATE field are documented and tagged to their retro deltas.
- Part 1 leaves the retros pending, adds a `changelog.d/` fragment, and `git status --short` is clean.

## Checks
`go test ./internal/lint/... ./internal/reportmodule/...` and `go vet` (from `go/`); bats
`tests/scaffold-module.bats`, `tests/lint-module-map.bats`, the touched lint-index / kit-links / doc-pin suites;
shellcheck on touched `.sh`; `gen-lint-index.sh --check`; `gen-retro-index.sh --check`;
`scripts/changelog-gate.sh --range db22d23e..HEAD`; `.githooks/pre-push` on the branch range.

## Test-first policy
M2 and M3 change behaviour: each gets a failing `go test` or bats run first (RED), recorded below with the failing
test names, then the implementation (GREEN). M0 and M1 are documentation: their check is structural (bats pins,
fold audit, generated-index `--check`).

## Progress
(updated after each work unit)

- M0 (route: delegated writer, this worker). The three retros were created with `toolbelt/new-retro.sh kit <slug>
  --date 2026-10-27 --deltas N` (5, 5, 8), which also set the kit envelope's `retro_pending: true` in
  `BUILD-STATE.src.md`, then filled with anonymized evidence.

## Next
Part 2 starts at M4. Read the delta mapping above; the retros hold the evidence and the targets.
