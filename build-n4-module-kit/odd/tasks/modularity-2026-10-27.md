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
- [x] M4 · strict rules (part 2): strict-new Δ3 (strict L3), Δ4 (strict lint-size), Δ5 (slot wall); gaps Δ1
  (model import direction), Δ5 (adapter decision budget, WARN). Landed as WU6 ba197046 and WU7 b48e9808.
- [x] M5 · front end, servlet, report (part 3): gaps Δ2 and Δ3 (rc-scan), Δ4 (dashboard JS skeleton), Δ6 (servlet
  router + handlers), Δ8 (report-module modularity summary), plus the module-map Δ3 report member. Landed as WU9
  3d5a9798, WU10 62cc8332, WU11 959be0c6 and WU12 5d2f2fba, after the part-2 review fix-forward WU8 2e261afb.
- [ ] M7 · retro D `2026-10-27-modularity-structure` (five items, see Next session).
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
  lint-structure, gen-lint-index, scaffold-module 83/83 ok; gen-lint-index --check 0. WU6 commit ba197046
  `feat(lint): strict L3 role packages and L3b model import direction` (+437/−10).
- WU7 = M4 part 2, strict-new Δ4 and Δ5, gaps Δ5 (route: delegated writer). RED: size goldens `strict-size` and
  `legacy-size` (no FAIL rows, no B-class or wrapped-signature method row, no adapter row) and verifymodule golden
  `strict-slotwall` (WARN and ALL PASS where FAIL and exit 1 were wanted); `nomap-size` recorded from the unchanged
  lint as the byte-identical baseline (it shows the old rule measures a wrapped signature from its `{` line: 9
  lines, no row, where the decl rule counts 12 from the name; the goldens run with `--max-class 20 --max-method 10`
  to stay small). GREEN: `size.mapped` (javasrc.Decls over every class, legacy WARN / strict
  FAIL, `adapter-branches` WARN for a Baja class importing a `model.` type with > 15 branch keywords);
  `slotWall` FAILs (and fails the gate) in a strict module. Docs: `types/logic.md` § Composition, METHODOLOGY size
  rule, lint-size and verify-module headers. Tests: `TestSlotWallStrict` (fixture PASS, 16-slot class FAIL in
  strict, WARN without the map); bats TC-MAP4 (every part of fresh scaffolds clean under lint-size; a long
  B-class method FAILs). Decision: the extended method rule applies only under a module map (legacy WARN), so
  map-less modules stay byte-identical (LSZ4 pins the pure-only rule there). WU7 commit b48e9808
  `feat(lint): strict size, slot wall and adapter branch budget` (+488/−14).

- Part 2 close checks (on b48e9808 + this document): full `memcap go test ./...` 121 packages ok, one FAIL —
  `reportmodule TestEveryLintIsAMemberOrListed` (the known WU2 gap above); bats 14 suites 235/235 ok (scaffold-module,
  lint-module-map, lint-structure, lint-size, verify-module, kit-links, gen-lint-index, guard-pins, kit-p1 control/
  web/tests pins, permissions-root, commissioning-verify, lint-machine-default-path); shellcheck clean on the five
  touched wrappers; gen-lint-index and gen-retro-index `--check` 0; strict fold audit 407/407 cited, 766 Δ checked,
  0 not cited; sweep-build-state 0; `changelog-gate --range e2b0e33e..HEAD` PASS (4 fragments).

- WU8 = part-2 review fix-forward, part 3 (route: delegated writer). RED: `TestAdvisoryMemberStrictFail` (an exit-1
  lint-size with FAIL rows printed `ERROR env fault (exit 1)`, the rows hidden) and size golden
  `adapter-static-import` (no `adapter-branches` row for `import static …model.Core.step;`); `adapter-boundary`
  (exactly 15 branches, and branch keywords inside a slotomatic region) passed on first run as a near-miss pin and
  was mutation-checked (budget 14 and dropping the region test both turn it red). GREEN: `advisoryMember` relays an
  exit 1 with a FAIL row as FAIL rows (R4-001, strict-new Δ4); `modelImport` accepts `import static`; size.go states
  the 0-based token / 1-based row line bases (R2-004); lint-size header FAIL row (R2-003); types/logic.md and
  METHODOLOGY size bullets consistent (R2-002); part-2 ledger open_issue current (R2-001), and its bare
  `<module-root>/MODULE-MAP.md` reference fixed (it failed kit-links L1 in a materialized BUILD-STATE.md). Checks:
  `go test` size, reportmodule ok; `go vet` ok; bats lint-size, report-module, guard-pins, gen-lint-index,
  kit-links 132/132; shellcheck clean; gen-lint-index --check 0. Commit 2e261afb
  `fix(report-module): relay a strict module's size FAIL rows as FAIL` (+194/−13).

- WU9 = M5 part 1, module-map Δ3 (report member) and gaps Δ8 (route: delegated writer). RED:
  `TestEveryLintIsAMemberOrListed` (lint-module-map removed from `notInReport`), `TestScriptsExist` (55 members),
  `TestModularityRow` and `TestSummarize` (stubs). GREEN: perModule `lint-module-map` (base-mode rows, failRows
  verdict) after lint-structure, then the uncounted `(module)  INFO  modularity  mode= concerns= unlisted= largest=`
  row (`modulemap.Summarize`, `size.HandLines`). The 104 report goldens that print the module members were
  re-recorded through the Go tool (a temporary, uncommitted rewrite hook in `golden.Runner`) and audited by script:
  every diff is +1 `PASS lint-module-map clean`, +1 INFO row, +24 real `WARN lint-module-map … MM1` rows in the
  `fx-` cases, and a summary line whose PASS rose by 1 and WARN by the MM1 rows, with FAIL/SKIP/ADVISORY, verdict
  and exit unchanged (104/104 OK). Checks: `go test` reportmodule, modulemap, size, lintindex, lint/rules ok;
  `go vet` ok; bats report-module, lint-module-map, gen-lint-index, guard-pins, kit-links 119/119; shellcheck clean;
  gen-lint-index --check 0. Commit 3d5a9798 `feat(report-module): run lint-module-map and print a modularity row`
  (+633/−110, of which ~440 lines are the re-recorded goldens).
- WU10 = M5 part 2, gaps Δ4 (route: delegated writer). RED: bats TC-MAP5 (`missing js/config.js`). GREEN: the
  MinimalDash fixture's `rc/` holds `js/config.js`, `js/services/api.js` (the only fetch: XHR header + abort
  timeout), `js/store.js` (poll, state, subscribe, `location.reload()` after `reloadAfterMs` without a good read),
  `js/components/status.js`, `js/pages/home.js`, `js/main.js`; `index.html` loads them in order with no inline
  script; the map's `frontend` row split into `frontend-data` and `frontend`. A fresh dashboard's report-module
  run loses its two WARN rows (`fetch-no-signal`, `lint-spa-poll-no-recovery`; the latter re-appears when the reload
  is removed, so the lint still reads store.js). Checks: bats scaffold-module 29/29, lint-vendor-floor, rc-scan,
  report-module, kit-links ok; `node --check` on every emitted js file ok; ESLint SKIP (not installed here). Commit 62cc8332 `feat(scaffold): emit a split rc/js skeleton for
  dashboards` (+165/−19).
- WU11 = M5 part 3, gaps Δ2 and Δ3 (route: delegated writer). RED: rcscan goldens `modmap-legacy`, `modmap-strict`
  (no rows; `modmap-none` passed first as the map-less byte-identical baseline), `TestJSFileSize` (the 401-line
  file case generated in Go, not stored as a golden), bats RC37 and the TC-MAP5
  positive control (a page fetch stayed exit 0). GREEN: `go/internal/rcscan` reads `lint.ModuleMode(root)`; under a
  map `fetch-outside-services` (a `fetch(` in a .js file or an inline script outside `rc/js/services/`) and
  `js-file-size` (> 400 lines, at line 1) WARN legacy (FAIL under --strict) and FAIL strict; no map runs neither.
  Decision: "outside services/" follows the retro's `js/services/` wording rather than only `services/api.js`, so
  a module may split services per domain. Mutation-checked: budget 399, dropping the allow marker and a strict
  module read as legacy each turn a golden red. Shim contract unchanged (same args, exits; two new row ids).
  Checks: `go test` rcscan, reportmodule ok; `go vet` ok; bats rc-scan 40/40, scaffold-module, guard-pins,
  gen-lint-index, kit-links, report-module ok; shellcheck clean; gen-lint-index --check 0. Commit 959be0c6
  `feat(rc-scan): fetch-outside-services and js-file-size under a module map` (+204/−1).
- WU12 = M5 part 4, gaps Δ6 (route: delegated writer). RED: `TestHandlerSize` (no row in any mode) and bats
  TC-MAP6 (a strict fresh dashboard with an 83-line servlet method exited 0). GREEN: `lint-servlet` `handler-size`:
  a method of a servlet file spanning more than 80 lines (name line to closing brace, javasrc.Decls) WARNs with no
  map or a legacy map (FAIL under --strict) and FAILs in a strict module. Decision: unlike lint-size's mapped rule
  and the rc-scan checks (map modes only, because a no-map WARN would change existing goldens), handler-size also
  WARNs with no map: every lint-servlet golden stays byte-identical (52/52), and it is the one rule that reaches
  the large legacy servlets of the audit. DUX1 in `types/dashboard.md` names the router + `ux/route/<Route>Handler`
  split. The fixture servlet keeps its two tiny inline branches (no handler class emitted). Gap: the wrapper header
  `build-n4-module-kit/toolbelt/lint-servlet.sh` (outside this part's edit surfaces) does not list handler-size yet.
  Checks: `go test` servlet ok (52 goldens unchanged); `go vet` ok; bats scaffold-module 30/30, lint-servlet,
  guard-pins, gen-lint-index, kit-links, report-module ok; gen-lint-index --check 0. Commit 5d2f2fba
  `feat(lint): servlet handler-size for the router-plus-handlers split` (+106/−2).
- Part 3 close checks (on 5d2f2fba + this document and the part-3 ledger fragment): full `memcap go test ./...`
  122 packages ok, 0 FAIL (`TestEveryLintIsAMemberOrListed` passes: lint-module-map is a member);
  `go test ./internal/lint/... ./internal/reportmodule/... ./internal/rcscan/... ./internal/verifymodule/...` ok;
  `go vet` size, modulemap, servlet, reportmodule, rcscan ok; bats 12 suites 340/340 ok (9 env skips: no
  niagara_home/JDK 8, no acorn/eslint) — scaffold-module, lint-module-map, lint-size, rc-scan, report-module,
  verify-module, lint-servlet, kit-links, gen-lint-index, guard-pins, lint-vendor-floor, lint-structure;
  shellcheck clean on lint-size, report-module, rc-scan; gen-lint-index and gen-retro-index `--check` 0; strict
  fold audit 407 folded / 407 cited / 0 uncited; sweep-build-state 0; `changelog-gate --range fe06e0be..HEAD`
  PASS (5 fragments).

## Next session
Part 3 stopped on 2026-10-27 after WU12 (5d2f2fba) and the part-3 ledger fragment; M0-M5 are done, the three retros
stay `review-status: pending`. Review candidate: 2e261afb..HEAD (five work units and this bookkeeping, base
fe06e0be).
- First, outside part 3's edit surfaces: document `handler-size` in the `toolbelt/lint-servlet.sh` header (a
  Checks row: WARN, FAIL in a strict module or under --strict; a `Mutation:` pin for TC-MAP6 or
  `TestHandlerSize`), then `gen-lint-index.sh --check`.
- M7 · retro D `2026-10-27-modularity-structure` (new, `toolbelt/new-retro.sh kit modularity-structure --date
  2026-10-27 --deltas 5`), five items:
  1. `scaffold-module add-concern <Name>` creates the concern folder, a pure class, its test, an adapter stub and the
     MODULE-MAP row.
  2. Fixed ROLE folders only (rt `model/`, `adapter/`; ux `route/`, `service/`); concern sub-folders under them have
     FREE per-module names; strict lint: every concern folder has exactly one MODULE-MAP row and vice versa, has
     tests, and no file sits in two concerns. Strict L3 (WU6) already fixes the role folders for Baja-free classes;
     `adapter/` for B-classes is not enforced yet.
  3. An explicit ux↔rt API contract file plus a contract test on each side.
  4. `module-map where <file|concern>` plus a runner that runs only that concern's tests.
  5. A legacy→strict migration guide that EXTENDS `METHODOLOGY.md:83` (opt-in per module, pure extraction first,
     slot changes last with migration + commissioning).
- M6 · close: flip the three retros (modularity-module-map, modularity-strict-new, modularity-gaps) to folded,
  strict fold audit 0 uncited (every Δ of the three now has an `[ev: retro … Δk]` tag in a kit doc), a
  `ledger.d/` fragment, `ci-local.sh -j 3`. Decide whether `fetch-outside-services`/`js-file-size` should also
  WARN with no map (today map modes only, to keep existing rc-scan goldens byte-identical) and whether the
  scaffold should emit a `ux/route/<Route>Handler` class.
