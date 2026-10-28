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
- [x] M1 · docs (WU1): module-map Δ1 (MODULE-MAP schema § in `types/structure.md`, METHODOLOGY conformance item),
  module-map Δ4 (agent rule in `skill/SKILL.md`), gaps Δ7 (test naming in `types/moduleTest.md`), module-map Δ5
  (`modularity` field in `BUILD-STATE.src.md`). Route: delegated writer. Checks: strict fold audit; kit-links and
  doc-pin bats.
- [x] M2 · Go, test-first (WU2): strict-new Δ1 `lint.ModuleMode(root)` with table tests; module-map Δ3 rule
  package `rules/modulemap` (MM1 to MM6), registration, wrapper `toolbelt/lint-module-map.sh`, goldens. Route:
  delegated writer. Checks: `go test`/`go vet` on `./internal/lint/...`; bats `tests/lint-module-map.bats`;
  `gen-lint-index.sh --check`; shellcheck.
- [x] M3 · scaffold, test-first (WU3, landed as WU5): module-map Δ2 (each fixture carries a strict `MODULE-MAP.md`, so the scaffold
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

- M0 committed 68accb9a (+258/−1). Sweep exit 0; strict fold audit 0 uncited / 0 not cited; gen-retro-index --check 0.
- M1 (route: delegated writer). Commit 8a8ab97f (+67). Bats kit-links, kit-p1-tests/control/web pins, install-skill,
  check-skill-drift, kit-k15, kit-libs-signing: 56/56 ok; sweep 0; strict fold audit 0 uncited. `skill/SKILL.md`
  changed (step 1e): reinstall + drift check owed (not run; the installed launcher already drifted before this).
- M2 (route: delegated writer). RED: `TestMapHeader`, `TestModuleMode`, `TestModularityString` (stubs), then
  `TestGolden` (14 of 15 cases), `TestUnreadablePart`, `TestMatchGlob`, `TestRootRulesOnAnEmptyRoot` failed. GREEN:
  `go test ./internal/lint/...` ok, `go vet` ok; bats lint-module-map, lint-empty-root, gen-lint-index, guard-pins,
  kit-links 49/49 ok; lint-guard-pins MATCH MMB1-MMB3; shellcheck clean; gen-lint-index --check 0. Decision: the
  type is `lint.Modularity` (`lint.Mode` already names the argument parse mode). Commit: see git log (WU2).
- Deferred from M2 to M5: the report-module perModule member. 104 of 123 report-module goldens print the
  lint-structure row and are recorded from the bash report-module at a98c3357; a failRows member adds a row to every
  one, so it is re-recorded once together with the modularity summary row (modularity-gaps Δ8).

- WU3/WU4 review (route: parent). RDD review of 6c34e20c..e2b0e33e APPROVED and acknowledged (lineage
  review-ed748625ba1a4a02) with five advisory findings, fixed forward in WU4.
- WU4 (route: delegated writer, part 2). Commit d7cdd7a1 `fix(lint): stop lint-module-map following symlinked
  directories` (+102/−23). RED: `TestCyclicDirSymlink` (test timeout: the walk looped through two `..` links),
  `TestSiblingDirAlias` (MM4 FAIL for `alias/B.java`). GREEN: the walk skips a symlinked directory (`e.Type()`
  is Lstat-based); `TestUnreadableMap` added (passed on first run: the read already failed closed, it was
  untested). R2-001 (exit-3 contract) and R2-002 (MM3 row) aligned in the wrapper header, the Spec doc comment and
  `types/structure.md`. Checks: `go test ./internal/lint/rules/modulemap/` ok, `go vet` ok; bats lint-module-map,
  gen-lint-index, kit-links 37/37 ok; shellcheck clean; gen-lint-index --check 0.
- WU5 = M3 (route: delegated writer). RED: `TC-MAP1` (no map in a fresh logic scaffold), `TC-MAP2` (no
  `model/ModuleALogic.java`). GREEN: a strict `MODULE-MAP.md` in each fixture module root (logic: control,
  station-test, plumbing; dashboard: control, api-routing, frontend, station-test, plumbing; driver: codec,
  network-device, points, discovery, point-manager, plumbing); `MinimalPanLogic`, `MinimalDashLogic`,
  `MinimalDriverCodec` and their tests moved to `…/model/` (public); `MinimalDashDispatch`, its test and
  `BMinimalDashServlet` moved to `…/ux/route/` (DUX1 kept package-private; ux `module-include.xml`, bajadoc
  package and DUX1 text updated); `go/internal/verifymodule/driver_test.go` follows the codec path. Pins updated:
  TC-DASH3, TC-DRV2, TC-DRV4, TC-DRV5, TC-DRV6, TC-DRV8. Checks: bats scaffold-module 26/26 (TC-MAP2, TC-DRV5 and
  TC-DRV8 ran the pure tests, not skipped), lint-structure, commissioning-verify, lint-machine-default-path,
  kit-links, lint-module-map, permissions-root: 120/120 ok; `go test` lint, verifymodule, javasrc,
  commissioningverify ok. Known base failure (not this WU, outside the edit surfaces):
  `reportmodule TestEveryLintIsAMemberOrListed` fails since WU2 because `lint-module-map` is neither a member nor in
  `notInReport` (`go/internal/reportmodule/completeness_test.go`); M5 owns the member. WU5 commit a3e39983
  `feat(scaffold): emit a strict module map and role packages for every type` (+198/−49).
- WU6 = M4 part 1, strict-new Δ3 and gaps Δ1 (route: delegated writer). RED: golden `strict-l3` (no strict row
  emitted); `legacy-l3` recorded from the unchanged lint (one advisory L3 WARN) as the byte-identical baseline.
  GREEN: `go/internal/lint/rules/structure/strict.go` — in a strict module, per file of each -rt/-ux/-wb/-se part:
  L3 FAIL for a Baja-free class (no `javax.baja`/`com.tridium` import, no `@NiagaraType`) outside its role package
  (rt/wb/se `model/`; ux `route/`, `service/`, `model/`), for a `model/` framework import (at the line) and for a
  `model/` class with no `<C>Test.java` under the part's `srcTest/`; L3b FAIL for a `model/` import of the
  module's own non-model package. The mode comes from `lint.ModuleMode(root)`; legacy and map-less modules keep
  `pureModel` unchanged. Docs: `types/structure.md` L3 row aligned with the code plus an L3b row; wrapper header.
  Bats TC-MAP3: fresh logic/dashboard/driver scaffolds have no L3 row, and a stray ux class and an untested model
  class FAIL. Checks: `go test ./internal/lint/...` ok (structure golden 44 cases), `go vet` ok; bats
  lint-structure, gen-lint-index, scaffold-module 83/83 ok; gen-lint-index --check 0.

## Next session
Session stopped at 16:00 CST on 2026-10-27 after WU2; WU3 was not started.
- M3 (WU3): put a strict `MODULE-MAP.md` in each fixture module root (`fixtures/<T>/<T>/MODULE-MAP.md`; the scaffold's
  copy + substitution then emits it, and TC3/TC-DASH1/TC-DRV1 diff it); move `MinimalPanLogic` and
  `MinimalDashLogic` (and their tests) into `…/model/` (public class + method; javadoc links to the B-class become
  plain text). Gap: moving `MinimalDriverCodec` breaks `go/internal/verifymodule/driver_test.go:17,141,144`, outside
  part 1's edit surfaces; `MinimalDashDispatch` is package-private by DUX1 and stays put until M4 decides. Driver map:
  the loopback station test covers network-device, points (with the -wb point manager) and discovery. Bats: a fresh
  scaffold of every type passes `lint-module-map` with 0 FAIL; a `What the scaffold emits today` bullet tagged
  module-map Δ2 and strict-new Δ2.
- M4, M5, M6 as listed under Tasks (M5 now also carries the report-module member of module-map Δ3).
- M7 · retro D `2026-10-27-modularity-structure` (new, five items):
  1. `scaffold-module add-concern <Name>` creates the concern folder, a pure class, its test, an adapter stub and the
     MODULE-MAP row.
  2. Fixed ROLE folders only (rt `model/`, `adapter/`; ux `route/`, `service/`); concern sub-folders under them have
     FREE per-module names (e.g. lighting `model/schedules/`, data center `model/cooling/`, UPS `model/battery/`); the
     kit never fixes concern names. Strict lint: every concern folder has exactly one MODULE-MAP row and vice versa,
     has tests, and no file sits in two concerns.
  3. An explicit ux↔rt API contract file plus a contract test on each side.
  4. `module-map where <file|concern>` plus a runner that runs only that concern's tests.
  5. A legacy→strict migration guide that EXTENDS `METHODOLOGY.md:83` (opt-in per module, pure extraction first,
     slot changes last with migration + commissioning).
