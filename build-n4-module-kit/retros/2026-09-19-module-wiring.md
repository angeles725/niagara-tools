<!-- review-status: folded -->
# 2026-09-19 · kit · module-wiring

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the module-wiring cluster.
**Delta count**: 7

## What happened
The kit taught Shadow for third-party bundling and scattered `api()`/`@AgentOn` mentions, but had
no single reference for how a module declares dependencies on other modules, registers its own
types, and calls across module boundaries. Cluster WIR-G1..G7 in the master register.

## Evidence
- Gradle configs → module.xml mapping: nmodsreflow gradle.kts + generated module.xml `[ev: code nmodsreflow gradle.kts]`
- Runtime cross-module call is module-agnostic (classloader bridges via api() dep): `[ev: corpus B802]`
- classloader isolation / uberjar embedding: `[ev: corpus B617]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | Gradle dep-config → module.xml impact table (nre/api/compileOnly/uberjar/moduleTestImplementation) | `types/module-wiring.md` §1 | `[ev: code nmodsreflow gradle.kts]` |
| Δ2 | uberjar() first-party fat-jar (vs Shadow) | `types/module-wiring.md` §1 | `[ev: code nmodsreflow gradle.kts]` |
| Δ3 | sibling-part api(project(":mod-rt")) → OEM vendor/version stamp | `types/module-wiring.md` §1 | `[ev: code nmodsreflow-ux module.xml]` |
| Δ4 | compileOnly(files($niagara_home/bin/ext)) for provided jars | `types/module-wiring.md` §1 | `[ev: code nmodsreflow gradle.kts]` |
| Δ5 | own-type + ordScheme registration via module-include.xml | `types/module-wiring.md` §2 | `[ev: corpus B35]` |
| Δ6 | agent-on XML-first vs @AgentOn equivalence | `types/module-wiring.md` §3 | `[ev: code nmodsreflow module.xml]` |
| Δ7 | cross-module refs: moduleName:TypeName ORD + Sys.getService + service: ORD | `types/module-wiring.md` §4 | `[ev: corpus B802]` |

## Lessons
- `api()` becomes a runtime `<dependency>`; `uberjar()`/`compileOnly()` do NOT — know which crosses the classloader boundary.
- A lib both api()'d and uberjar()'d = a classloader conflict; an `<on type>` typo = a silent UI failure.

---
**Status**: FOLDED into `types/module-wiring.md` (2026-09-19).
