<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-27 · kit · modularity-module-map

**Session**: niagara-tools modularity campaign part 1, branch `docs/fold-r10-retros` from origin/main 6c34e20c
(read-only audit of the kit and four client modules; proposal approved by the parent session)
**Delta count**: 5

## What happened
A read-only audit asked why modules built with the kit grow into a few very large files. The kit tells a builder
what to build (types/*.md) and gates each file on its own (lint-structure, lint-size, rc-scan), but nothing in a
module says which files hold which concern, which tests cover it, or where a change belongs. An agent therefore
opens the largest file that mentions the symbol and adds to it. Four client modules (A, B, C, D) all show it: one
adapter class carries most of the behaviour, tests are named by concern with no matching source class, and a
dashboard front end is one HTML file. The kit's own fixtures do not show the alternative either.

The fix proposed here is a module-root `MODULE-MAP.md`: a header that says whether the module is held to the
strict rules (`modularity: strict`, written by the scaffold) or is a legacy module (`modularity: legacy` plus a
reason), and one table row per concern naming its part, files, tests and "change here when". A new lint checks the
map against the tree, the scaffold writes one per type, and the agent rule says to read it first.

## Evidence
- Client module A, `-rt`: the adapter class has 1362 hand-written lines and 112 slots; its `execute()` method is
  264 lines. Its pure core has 671 lines, three methods over 80 lines, and a `step()` of 220 lines whose signature
  is wrapped over 6 lines.
- Client module B, `-rt`: the unit adapter has 718 lines, 30 slots and 88 branches; `changed()` is 96 lines. The
  module has 18 tests named by concern and no source class matching any of those names.
- Client module C: panel components with 120 and 72 slots; the `-ux` servlet is 627 lines (one write handler is
  177 lines); `buildEquipmentResponse` in its reader is 247 lines; the front end is a single `index.html` of 3260
  lines (2432 lines of script, 610 of CSS, 87 functions, 10 fetch call sites), with no services, store, components
  or pages.
- Client module D, `-ux`: 54 JS files, 11 of them over 500 lines (the largest 4049), 20 files call `fetch`; the
  servlet has 1609 lines and 28 `handle*` methods. Two other single-page apps have an `index.html` of 3298 and 2300
  lines.
- Modules A, B and C: 0 of 4 parts have a `model/` package. Five classes across the modules have more than 15
  slots.
- Hosts checked in the kit at origin/main 6c34e20c: `go/internal/lint/lint.go:121` (RootRule), `:204`
  (Spec.ModuleRoot), `:234` (Spec.BarePathRow); `go/internal/lint/rules/rules.go` `All()`;
  `go/internal/reportmodule/members.go:104` (lint-structure as a perModule member); `toolbelt/scaffold-module.sh`
  (fixture copy, then the station-test overlay); `BUILD-STATE.src.md:40` (`ui_profile`, the precedent for a
  declared per-module field with an honest `unknown`).
- Rejected alternatives for "is this module new": `.deploy-baseline/` (appears only at the first deploy), git
  history (report-module is VCS-free, `toolbelt/report-module.sh:64`), and the BUILD-STATE envelope as the source
  of truth (it lives in the kit repo; a lint must read the module tree alone).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Define `MODULE-MAP.md` at the module root: a `modularity: strict` or `modularity: legacy` header (`legacy-reason:` for legacy), then a `Concern | Part | Files | Tests | Change here when` table; Java files listed one per entry, front-end `src/rc` entries may be directory globs; METHODOLOGY conformance item | `types/structure.md` § Module map; `METHODOLOGY.md` § Conformance rules | `[ev: code go/internal/lint/lint.go]` |
| Δ2 | `scaffold-module.sh` emits a per-type `MODULE-MAP.md` with `modularity: strict` for logic, dashboard and driver, and a fresh scaffold of each type passes `lint-module-map` with 0 FAIL | `toolbelt/scaffold-module.sh`; `fixtures/*`; `tests/scaffold-module.bats` | `[ev: code build-n4-module-kit/toolbelt/scaffold-module.sh]` |
| Δ3 | New `lint-module-map` (MM1 no map WARN; MM2 header FAIL; MM3 listed path missing FAIL; MM4 every part source file in exactly one concern, unlisted FAIL strict / WARN legacy, listed twice FAIL; MM5 every non-plumbing concern names an existing srcTest path, FAIL strict / WARN legacy; MM6 a srcTest class under no concern WARN), a Go RootRule with a toolbelt wrapper and a report-module perModule member | `go/internal/lint/rules/modulemap`; `go/internal/lint/rules/rules.go`; `toolbelt/lint-module-map.sh`; `go/internal/reportmodule/members.go` | `[ev: code go/internal/reportmodule/members.go]` |
| Δ4 | Agent rule: read `MODULE-MAP.md` first; edit only the files of the concern being changed; a new file adds its row in the same change | `skill/SKILL.md` § Execution Steps | `[ev: code build-n4-module-kit/skill/SKILL.md]` |
| Δ5 | Mirror `modularity` in the module's BUILD-STATE envelope for orientation only (the lint never reads it) | `BUILD-STATE.src.md` § field doctrine | `[ev: code build-n4-module-kit/BUILD-STATE.src.md]` |

## Lessons
- A per-file gate cannot see where a change belongs; only a module-level map can say which file owns a concern.
- Mark a module new or legacy in the module tree itself: anything outside it (deploy baselines, git, the kit
  ledger) is either late, VCS-bound or invisible to a VCS-free lint.
- New modules get the strict rule from the scaffold; existing modules get warnings until they opt in.
