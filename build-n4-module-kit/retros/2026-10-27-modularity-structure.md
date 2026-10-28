<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-27 · kit · modularity-structure

**Session**: niagara-tools modularity campaign part 3b, branch `docs/fold-r10-retros` (after WU12 5d2f2fba; parent
session decisions for the five structure items)
**Delta count**: 5

## What happened
The first three modularity retros (modularity-module-map, modularity-strict-new, modularity-gaps) gave every module a
map, strict rules for new modules and front-end and servlet checks. Reviewing the result before the close, the parent
session found five structure gaps that keep a module from staying modular as it grows: adding a concern is a manual
edit of several files and the map; the role folders are fixed but nothing checks the concern sub-folders under them;
the ux front end and the rt servlet agree on response shapes only by convention; nothing answers "which concern owns
this file and which tests cover it" from the command line; and the existing-module rule (METHODOLOGY.md:83, do not
restructure deployed modules unless asked) has no path from legacy to strict when a restructure IS requested.

## Evidence
- `toolbelt/scaffold-module.sh`: only emits a whole module; a new concern (folder, pure class, test, map row) is
  written by hand, so its map row is the step most often skipped. `[ev: code build-n4-module-kit/toolbelt/scaffold-module.sh]`
- `go/internal/lint/rules/modulemap/modulemap.go`: MM4 checks file ownership per file; no check relates a concern
  sub-folder (rt `model/<concern>/`) to one map row, so one folder may be split over two rows or one row may span
  two folders. `[ev: code go/internal/lint/rules/modulemap/modulemap.go]`
- `fixtures/MinimalDash`: `MinimalDashDispatch` answers `{"ok":true}` and `js/services/api.js` reads it, but no file
  names that shape and no test on either side fails when one side changes it. `[ev: code build-n4-module-kit/fixtures/MinimalDash]`
- `toolbelt/run-pure-test.sh` runs one named test class; finding a concern's tests means reading the map by eye.
  `[ev: code build-n4-module-kit/toolbelt/run-pure-test.sh]`
- `METHODOLOGY.md:83` forbids restructuring a deployed module to meet the new rules unless explicitly requested,
  and requires a biting safety-invariant suite green UNCHANGED after a requested move; it gives no order of steps.
  `[ev: code build-n4-module-kit/METHODOLOGY.md]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `scaffold-module add-concern <Name> --root <module-root> [--part rt\|ux\|wb\|se] [--adapter]`: creates the concern sub-package under the part's role folder, a pure class and its `<Class>Test`, an adapter stub only on `--adapter`, and appends the MODULE-MAP row; refuses an existing concern (Go, shimmed) | `toolbelt/scaffold-module.sh`; `go/cmd/module-map` | `[ev: code build-n4-module-kit/toolbelt/scaffold-module.sh]` |
| Δ2 | Role folders fixed, concern names free: rt `model/`, `adapter/`; ux `route/`, `service/`; wb/se `model/`; lint-module-map MM7: each concern sub-folder under a role folder maps to exactly one row and a non-plumbing row's files sit in at most one concern folder (WARN legacy, FAIL strict) | `types/structure.md` § Module map; `go/internal/lint/rules/modulemap` | `[ev: code go/internal/lint/rules/modulemap/modulemap.go]` |
| Δ3 | An explicit ux↔rt API contract: `<M>-ux/src/rc/js/contract.json` describes the servlet's response shapes; a Java srcTest asserts the servlet side and a JS spec asserts the consumer side; emitted by the dashboard scaffold | `types/dashboard.md`; `fixtures/MinimalDash` | `[ev: code build-n4-module-kit/fixtures/MinimalDash]` |
| Δ4 | `module-map where <file\|concern> --root <module-root>` prints the concern, its files and tests; `module-map tests <concern> --root <module-root>` runs only that concern's pure JUnit tests (run-pure-test.sh) and JS specs | `go/cmd/module-map`; `toolbelt/module-map.sh` | `[ev: code build-n4-module-kit/toolbelt/run-pure-test.sh]` |
| Δ5 | A legacy→strict migration guide that EXTENDS METHODOLOGY.md:83: opt-in per module; map as `legacy` first, then pure extraction into `model/<concern>/` with tests (no slot change), per-concern split, flip to `strict`, slot recomposition last with a persisted-slot migration plan and commissioning | `types/structure.md`; `skill/SKILL.md` | `[ev: code build-n4-module-kit/METHODOLOGY.md]` |

## Lessons
- A structure the kit wants kept needs a generator as well as a lint: `add-concern` makes the mapped shape the
  cheapest one to write.
- Fix the role, not the vocabulary: role folders are kit-wide, concern names belong to the module's domain.
- A cross-part data shape is a contract only when a file names it and a test on each side reads that file.
- A migration rule that only forbids needs an ordered path for when the move is requested.
