<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 24 -->
# 2026-11-15 · kit · doc-coverage-audit

**Session**: an audit of the kit's own documents against its code: two module shapes the guides prescribe that the structure lint rejected, seven places where a document contradicted the code, and the coverage the eight module kinds still lack.
**Delta count**: 24

## What happened
The kit documents eight kinds of module. Built exactly as `types/theme.md` and `types/utility-lib.md` prescribe
them, a theme module and a utility library both FAILed `lint-structure`: a theme is a `-ux` part with no Java and
no palette, which is L9's empty skeleton, and a library is an `-rt` part with no palette, which is L5's missing
palette. No exemption existed. One was added that is declared, never inferred: an optional `type: theme` or
`type: library` line in the header of the module's `MODULE-MAP.md`, the file that already carries the module's
`modularity:` mode. An undeclared part FAILs as before.

Seven documents disagreed with the code or with each other, and the code was right in each: where the station
test runner runs, which Spanish level names the console triage reads, which rules the structure lint has, whether
L10 is a WARN or a FAIL, the block range of the corpus index, and the checks the jar gate runs. Each now states
what the code does, and two Go tests fail when the rule table or the gate's header drifts from the code again.

The audit also listed what the eight kinds still lack. Those rows are open below for other writers.

## Evidence
- A theme tree of five files (`<name>-ux.gradle.kts`, `module-include.xml` with `<types/>`, `nss/theme.nss`,
  `ux/theme.css`, `hx/theme.css`) gave `L9: empty skeleton artifact`, exit 1; a library tree of three files
  (`<name>-rt.gradle.kts`, `module-include.xml`, `module-permissions.xml`) gave `L5: module.palette missing for -rt
  profile`, exit 1. Both observed as failing golden cases before the change. `[ev: 9812dc71]`
- The lint built from the base and from the change printed the same rows and exit code over the three kit fixtures
  and the seven trees of `tests/fixtures/lint-structure`. `[ev: 9812dc71]`
- The L5 text is pinned by four golden files of `go/internal/reportmodule/testdata`, so the "how to declare" hint
  went into the undeclared `-ux` L9 row only. `[ev: go/internal/reportmodule/testdata/fx-agent-on.txt]`
- The kit fixtures are scaffold templates with a gradle wrapper, each named by a scaffold test; nothing walks the
  directory, and a fixture there is expected to build with gradle, which this session could not prove offline. The
  two shapes are golden cases of the lint. `[ev: go/internal/scaffoldmodule/types.go]`
- The structure lint reports sixteen rule IDs (L1 to L7, L3b, L9 to L17); the guide's table held twelve and one
  the lint does not run (L8, the gate's `signed`). `[ev: b0c6667c]`
- The jar gate runs twenty-one checks; its header listed fourteen. `[ev: go/internal/verifymodule/checks.go]`
- The console triage matches seven Spanish level names in UTF-8 and Latin-1 (`isLevel`); the guide said three and
  carried a deferred marker for the fix. `[ev: go/internal/triageconsole/scan.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-15. A1: a module declares `type: theme` or `type: library` in its `MODULE-MAP.md` header; a declared theme's zero-Java `-ux` part passes L9 when it holds a `.nss` file, a declared library's `-rt` part passes L5 with no palette, and an undeclared part FAILs as before, the `-ux` L9 row saying how to declare | `go/internal/lint/rules/structure/kind.go` · `kindscan.go` · `types/structure.md` § Module map · `types/theme.md` § 7 · `types/utility-lib.md` § 6 | `[ev: 9812dc71]` |
| Δ2 | APPLIED 2026-11-15. A2: the minimal theme and the minimal Shape A library are checked on every test run, as golden cases of the structure lint (declared, undeclared, wrong kind, a theme with no `.nss`, a library with an empty palette); they are not kit fixtures, which must build with gradle (Δ10) | `go/internal/lint/rules/structure/testdata/kind-*.txt` · `kind_test.go` | `[ev: 9812dc71]` |
| Δ3 | APPLIED 2026-11-15. B1: the three documents say one thing about station tests: a native Windows install, no running station, and from WSL `run-niagara-test.sh run` through the Windows command shell | `METHODOLOGY.md` · `BUILD-LOOP.md` § 4.a · `types/moduleTest.md` | `[ev: 18d6da5b]` |
| Δ4 | APPLIED 2026-11-15. B2: the diagnostics guide states the seven Spanish level names the console triage matches and names the code and tests that pin them; the deferred marker is closed | `types/diagnostics.md` § 3 | `[ev: 18d6da5b]` |
| Δ5 | APPLIED 2026-11-15. B3: the structure guide's title, table and every citation name the rules the lint runs (L1 to L17, L8 marked as the gate's) | `types/structure.md` · `METHODOLOGY.md` · `skill/SKILL.md` | `[ev: 9812dc71]` |
| Δ6 | APPLIED 2026-11-15. B4: L10 is stated as the code has it in both documents: a WARN for the four host build-location keys in a client tree, a FAIL for them inside the kit's fixtures and templates and for `user_home` anywhere | `types/structure.md` · `BUILD-LOOP.md` § 1.a | `[ev: 18d6da5b]` |
| Δ7 | APPLIED 2026-11-15. B5: the skill no longer hard-codes the block range of the corpus index | `skill/SKILL.md` | `[ev: 18d6da5b]` |
| Δ8 | APPLIED 2026-11-15. B6: the gate's header lists every check it runs, in run order | `toolbelt/verify-module.sh` | `[ev: 18d6da5b]` |
| Δ9 | APPLIED 2026-11-15. B7: a Go test fails when the rule table of the structure guide and the rules the lint reports differ, and one when the gate's header and its registered checks differ | `go/internal/lint/rules/structure/doc_test.go` · `go/internal/verifymodule/header_test.go` · `checks.go` | `[ev: b0c6667c]` |
| Δ10 | A new module of the wb view, station service, utility library, theme and cloud connector kinds starts from a scaffold type or a fixture that builds and passes the lints, as the logic, dashboard and driver kinds do | `go/internal/scaffoldmodule/types.go` · `fixtures/` | `[ev: go/internal/scaffoldmodule/types.go]` |
| Δ11 | APPLIED 2026-11-15. `types/diagnostics.md` § 4 (thread dump by three routes and how to read it, remote debugger, launcher flags). The diagnostics guide tells an author how to take and read a thread dump, attach a remote debugger, and which launcher flags a station and Workbench accept | `types/diagnostics.md` | `[ev: build-n4-module-kit/types/diagnostics.md]` |
| Δ12 | APPLIED 2026-11-15. In `types/wb-widgets.md` (JS toolchain section) with a pointer from `types/dashboard.md`; the edit-and-reload mode is moduledev, webdev serves the unminified files, and both are documented. A front-end author can run a `-ux` module in webdev mode and knows what the JS build does and produces | `types/frontend-standard.md` · `types/dashboard.md` | `[ev: build-n4-module-kit/types/frontend-standard.md]` |
| Δ13 | APPLIED 2026-11-15. `types/wb-widgets.md` § Hx view authoring. An author can write an Hx view from the kit alone: the class, its registration and its update cycle | `types/wb-widgets.md` · `types/dashboard.md` | `[ev: build-n4-module-kit/types/wb-widgets.md]` |
| Δ14 | APPLIED 2026-11-15. `types/wb-widgets.md` § Custom-painted Workbench widget and § Workbench tool. An author can write a custom-painted widget and a Workbench tool from the kit alone | `types/wb-widgets.md` | `[ev: build-n4-module-kit/types/wb-widgets.md]` |
| Δ15 | APPLIED 2026-11-15. `types/theme.md` § 8-10; the flow starts from a copy of a shipped theme, as the developer guide prescribes. The theme guide gives the official theme authoring flow from an empty directory to a theme a user can pick | `types/theme.md` | `[ev: build-n4-module-kit/types/theme.md]` |
| Δ16 | A cloud connector has a test recipe and a checklist, and its two owed lints (G87 `mqtt-tls-port`, G88 `rdbms-dialect-descriptor`) exist | `types/cloud-connector.md` · `go/internal/lint/rules/` | `[ev: build-n4-module-kit/types/cloud-connector.md]` |
| Δ17 | APPLIED 2026-11-15. `types/service.md`, routed from `skill/SKILL.md`. A station service has a guide of its own: the service type, its lifecycle, its registration and its checks | `types/` (new card) | `[ev: build-n4-module-kit/skill/SKILL.md]` |
| Δ18 | APPLIED 2026-11-15. `types/security.md` § 5.2-5.3; how a third-party vendor gets a certificate and issues licence files stays open (Δ24). An author can license a feature of their own module: the feature name, the check and what an unlicensed station does | `types/security.md` · `types/distribution.md` | `[ev: build-n4-module-kit/types/security.md]` |
| Δ19 | APPLIED 2026-11-15. `types/distribution.md` § 10a with a row in `types/platform.md` § 2; `BUILD-LOOP.md` is unchanged. Removing a module from a station follows a checklist: what refers to it, what to remove first and how to confirm the station starts clean | `types/distribution.md` · `BUILD-LOOP.md` | `[ev: build-n4-module-kit/types/distribution.md]` |
| Δ20 | PARTLY APPLIED 2026-11-15. The title states the real range (B5–B1420) and 22 rows were added for the blocks the 2026-11-15 sections cite; left: a sweep of every block cited by the other cards (B1146 and B1254 among them) against the index. The corpus index holds an entry for every block the kit's guides cite, and its title states the range it reaches (it says B1258 and cites blocks beyond it) | `corpus-index.md` | `[ev: build-n4-module-kit/corpus-index.md]` |
| Δ21 | Each of the eight steps the skill requires and no tool reads either gets a check or is marked in the skill as unchecked: the behaviour decisions table, the design-shard checklist or its waiver, the checks-skipped section, the priority tier, preview before compiling, corpus search first, console triage after a reload, and a full station backup before a runtime install | `skill/SKILL.md` · `BUILD-LOOP.md` · kit lints | `[ev: build-n4-module-kit/skill/SKILL.md]` |
| Δ22 | The undeclared L5 row says how to declare a library, as the `-ux` L9 row does for a theme: the four report goldens that pin the L5 text are re-recorded with it; the header of `toolbelt/lint-structure.sh` states the two declared kinds and the bats title that says "L1-L11" names the real range | `go/internal/lint/rules/structure/kind.go` · `go/internal/reportmodule/testdata/` · `toolbelt/lint-structure.sh` · `tests/lint-structure.bats` | `[ev: go/internal/reportmodule/testdata/fx-agent-on.txt]` |
| Δ23 | The scaffold writes the `type:` line for a kind that needs one, and `lint.MapHeader` returns the declared kind so the structure lint and the module-map lint read the header with one reader; the improvement-candidates rows that still mark the Spanish level names as owed are closed | `go/internal/lint/paths.go` · `go/internal/scaffoldmodule/` · `odd/tasks/kit-improvement-candidates-2026-10-15.md` | `[ev: go/internal/lint/paths.go]` |
| Δ24 | Research gaps the documents now mark as open are answered or filed as niagara-research gaps: how a third-party vendor obtains a vendor certificate and issues licence files (`types/security.md` § 5.2); how to pass the debugger agent option to a station the platform daemon starts (`types/diagnostics.md` § 4.2); whether a Software Manager uninstall stops the station (`types/distribution.md` § 10a); the console verbs and launcher flags on 4.15 (niagara-research B1296-G1) and the 4.15 service start and stop code (`types/service.md`); the official switch-off procedure of webdev and moduledev modes (`types/wb-widgets.md`) | `types/security.md` · `types/diagnostics.md` · `types/distribution.md` · `types/service.md` · `types/wb-widgets.md` | `[ev: build-n4-module-kit/types/security.md]` |

## Lessons
- A lint that judges a shape by the files it lacks needs a declared exception before a guide may prescribe that
  shape; build the guide's smallest example and run the lint on it when the guide is written.
- A list of rules or checks written by hand in a document drifts from the code within a few changes; a test that
  reads both is cheaper than the audit that finds the drift.
- A row text another package's golden files pin cannot change inside one unit; say so and leave the row open.
- A marker for owed work is closed in the same change that does the work, or it outlives it.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-15-doc-coverage-audit.md | kit | 2026-11-15 | pending | 24 |`
