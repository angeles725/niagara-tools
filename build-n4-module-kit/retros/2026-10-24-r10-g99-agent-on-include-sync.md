<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-g99-agent-on-include-sync

**Session**: niagara-research quality round 10, branch `r10/g99-agent-on-include-sync` from origin/main 1be84fd1
**Delta count**: 1

## What happened
The lint half of retro `n4-course-handson-slotomatic-authoring` Δ5 (issue #473) was owed as Go lint G99: javac's
annotation processor never rewrites a `module-include.xml` `<type>` line whose existing `<agent>` elements all
match, and a line with no `<agent>` matches any annotation (`ModuleInclude.entryMatches`, nre 4.14.0.162 vineflower
`javax/baja/nre/annotations/processors/ModuleInclude.java:376-415`), so an `@AgentOn` added later never reaches the
file. `lint-agent-on-shape.sh` checks only the `module:Type` shape of each `<on type>`. The new rule reads Java as
`javasrc.Tokens` (a commented-out or quoted `@AgentOn` is not one) and the include as `xmlsrc` XML. Real trees: the
six `@AgentOn` part sources (Cliente `Apillm-wb` x2, `chihuahua-wb`; course `envCtrlDriver-wb` x2,
`typeExtensionDemo-ux`) are clean, and each FAILs once its `<agent>` blocks are stripped from a scratch copy.

## Deltas
| Δ | Change | Where | Evidence |
|---|---|---|---|
| Δ1 | New `toolbelt/lint-agent-on-include-sync.sh` (`n4lint lint-agent-on-include-sync`, rule `go/internal/lint/rules/agentonincludesync`, TreeRule over a module root): FAIL a part `src/` type whose `@NiagaraType` `@AgentOn` types are not all `<on type>` entries of its `<type class>` line, a class with no line, or a part with no `module-include.xml`; exit 3 for usage, a walk error, a malformed file or a root with none. Wired into `report-module.sh` step 8b (RM9b). | `toolbelt/lint-agent-on-include-sync.sh` + `tests/lint-agent-on-include-sync.bats` (AOIS1 Bonus Lab fixture, AOIS2) + `types/module-wiring.md` § 2 | `[ev: retro n4-course-handson-slotomatic-authoring Δ5]` |
