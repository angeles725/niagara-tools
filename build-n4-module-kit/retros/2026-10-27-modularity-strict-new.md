<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-27 · kit · modularity-strict-new

**Session**: niagara-tools modularity campaign part 1, branch `docs/fold-r10-retros` from origin/main 6c34e20c
(read-only audit of the kit and four client modules; proposal approved by the parent session)
**Delta count**: 5

## What happened
The same audit as retro modularity-module-map asked why the existing structure and size gates did not stop the
large classes found in client modules A to D. Each gate turned out to be advisory or blind on the shape that
mattered: L3 only checks a `model/` package that exists, so a module without one passes it vacuously; lint-size is
WARN-only and its method rule skips B-classes and wrapped signatures; the slot-wall check is a PASS/WARN row. The
kit's own fixtures keep their pure classes in the root package, so they would fail a strict L3 themselves.

The proposal: a module the scaffold marks `modularity: strict` gets these checks as FAIL rows, and a legacy module
keeps today's WARN rows. A per-file lint needs a way to know which mode applies to a file, hence a walk-up helper.

## Evidence
- `go/internal/lint/rules/structure/profile.go:111-141` (`pureModel`): L3 WARNs only for a `model/` directory that
  imports `javax.baja.*`; `types/structure.md:106` promises "a `srcTest` test per class AND zero `import
  javax.baja`". Modules A, B and C have no `model/` package in any of 4 parts, so L3 never runs on them.
- `toolbelt/lint-size.sh:23-28`: only pure classes get the method rule, and a signature wrapped over several lines
  before `{` is not seen. Of the 8 long methods measured in modules A to C, the rule catches 2 or 3 (A's `step()`
  signature spans 6 lines; A's `execute()` and B's `changed()` sit in B-classes).
- `go/internal/lint/rules/size/size.go:51` (`Severity` is Warn) and `:62-91` (`Check`).
- `go/internal/verifymodule/srcmore.go:275`: the slot-wall row is PASS or WARN at >15 `@NiagaraProperty`;
  `types/logic.md:206` says to compose above about 12 to 15 flat slots. Five classes in the client modules exceed 15.
- Kit fixtures: `MinimalPanLogic`, `MinimalDashLogic` and `MinimalDriverCodec` are Baja-free and live in the
  module's root package (`fixtures/*/…/src/com/angeles/<Mod>/`), not in `model/`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `lint.ModuleMode(root)`: walk up from a path to the first `MODULE-MAP.md`, stopping at `.git`, and return strict, legacy or none, so a per-file lint can pick its severity | `go/internal/lint/paths.go` | `[ev: code go/internal/lint/paths.go]` |
| Δ2 | Move the fixtures' Baja-free classes into a `model/` package (with Δ3, so the fixtures pass strict L3) | `fixtures/MinimalPan`; `fixtures/MinimalDash`; `fixtures/MinimalDriver` | `[ev: code build-n4-module-kit/fixtures/MinimalPan]` |
| Δ3 | Strict L3: a Baja-free class outside `model/` FAILs, a `model/` class without a srcTest FAILs, a Baja import in `model/` FAILs; `types/structure.md` L3 row aligned | `go/internal/lint/rules/structure/profile.go`; `types/structure.md` § L1–L11 | `[ev: code go/internal/lint/rules/structure/profile.go]` |
| Δ4 | Strict lint-size: a class over 800 hand-written lines FAILs; the method rule covers B-class methods and wrapped signatures; report-module mode-aware | `go/internal/lint/rules/size/size.go`; `go/internal/javasrc`; `go/internal/reportmodule/members.go` | `[ev: code go/internal/lint/rules/size/size.go]` |
| Δ5 | Strict slot wall: more than 15 `@NiagaraProperty` in one class FAILs | `go/internal/verifymodule/srcmore.go`; `types/logic.md` § Composition | `[ev: code go/internal/verifymodule/srcmore.go]` |

## Lessons
- A conformance rule that is only checked when its subject exists (a `model/` package) passes every module that
  skipped the subject; check for the subject first.
- A size rule that skips the biggest classes by construction measures the wrong population.
- The kit's own fixtures must pass the strictest rule the kit ships, or the scaffold teaches the opposite.
