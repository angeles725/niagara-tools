<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 8 -->
# 2026-10-27 · kit · modularity-gaps

**Session**: niagara-tools modularity campaign part 1, branch `docs/fold-r10-retros` from origin/main 6c34e20c
(read-only audit of the kit and four client modules; proposal approved by the parent session)
**Delta count**: 8

## What happened
Beyond the module map (retro modularity-module-map) and the strict mode for new modules (retro
modularity-strict-new), the audit found gaps where no kit rule says how a concern is split: a `model/` package may
import the module's own adapter packages, nothing limits the size of a front-end JS file or where `fetch` is
called, the dashboard scaffold ships a single `index.html` with no JS structure, a servlet grows one handler method
per route, and a B-class adapter accumulates decisions that belong in its pure peer. Test naming and the per-module
report summary are the two small closing pieces.

## Evidence
- Client module C `-ux`: one `index.html` of 3260 lines, 10 fetch call sites in it, no services/store/components/
  pages; servlet 627 lines with a 177-line write handler. Client module D `-ux`: 20 of 54 JS files call `fetch`, 11
  files are over 500 lines (the largest 4049), the servlet has 1609 lines and 28 `handle*` methods.
- Client module B `-rt`: 88 branches in the unit adapter, whose pure peer exists; tests are named by concern
  with no matching source class.
- `toolbelt/rc-scan.sh:56`: the only size check is `inline-block-size` (an inline `<script>`/`<style>` block over
  300 lines); no rule reads a `.js` file's length or its `fetch` sites.
- `types/dashboard.md:66-67` (DUX1): the servlet delegates to a pure router, but nothing says where each route's
  handler lives.
- `fixtures/MinimalDash/…/MinimalDash-ux/src/rc/` holds only `index.html`.
- `go/internal/lint/rules/structure/profile.go` has no import-direction check for `model/`.
- `go/internal/reportmodule/report.go`: the summary counts rows per verdict; nothing reports a module's
  modularity mode.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `model/` must not import the module's non-model packages (L3b), FAIL in strict | `go/internal/lint/rules/structure/profile.go` | `[ev: code go/internal/lint/rules/structure/profile.go]` |
| Δ2 | rc-scan `fetch-outside-services`: a `fetch(` outside `js/services/` WARNs, FAILs in strict | `toolbelt/rc-scan.sh` and its Go port | `[ev: code build-n4-module-kit/toolbelt/rc-scan.sh]` |
| Δ3 | rc-scan `js-file-size`: a `.js` file over about 400 lines WARNs, FAILs in strict | `toolbelt/rc-scan.sh` and its Go port | `[ev: code build-n4-module-kit/toolbelt/rc-scan.sh]` |
| Δ4 | The dashboard scaffold emits a JS skeleton: `js/config.js`, `js/services/api.js`, `js/store.js`, `js/components/`, `js/pages/`, `js/main.js` | `fixtures/MinimalDash` `-ux` `src/rc` | `[ev: code build-n4-module-kit/fixtures/MinimalDash]` |
| Δ5 | Adapter decision budget: a B-class with a pure peer and more than about 15 hand-written branches WARNs (WARN only until calibrated) | `go/internal/lint/rules/size` sub-check | `[ev: code go/internal/lint/rules/size/size.go]` |
| Δ6 | Servlet = router plus one handler class per route group (`ux/route/<Route>Handler`); a handler body over 80 lines inside the servlet WARNs, FAILs in strict | `types/dashboard.md` § DUX1; `go/internal/lint/rules/servlet` | `[ev: code build-n4-module-kit/types/dashboard.md]` |
| Δ7 | Test naming: `<PureClass>Test` for a pure class, `<Concern>…Test` for a concern; MM5 of lint-module-map enforces that each concern has one | `types/moduleTest.md` § What to test | `[ev: code build-n4-module-kit/types/moduleTest.md]` |
| Δ8 | report-module prints a modularity summary row per module (mode, concerns, unlisted files) | `go/internal/reportmodule/report.go` | `[ev: code go/internal/reportmodule/report.go]` |

## Lessons
- A split that no rule names is not kept: every concern boundary the kit wants needs a home in the scaffold and
  a check in a lint.
- Front-end structure decays fastest because the kit gates it least; give it the same map and size rules as Java.
