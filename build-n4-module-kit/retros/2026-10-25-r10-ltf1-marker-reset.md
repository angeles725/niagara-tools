<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-25 · kit · r10-ltf1-marker-reset

**Session**: niagara-tools quality round 10 (Ayudante2), lint-link-target-flags LTF1, branch
`r10/ltf1-marker-reset` from origin/main ba3b7aa4
**Delta count**: 2

## What happened
A facade whose class javadoc says a READONLY slot is a link target passed LTF1. In
`go/internal/lint/rules/linktargetflags` `Check`, any line that is neither a comment nor a property cleared the
comment block, and `@NiagaraType` sits between the class javadoc and the `@NiagaraProperty` lines, so the javadoc
never headed `status`. The kit's own `fixtures/MinimalDash` BMinimalDash shows the shape: "one READONLY display slot
({@code status}) that the integrator links from a control module's output point", then `@NiagaraType`, then
`status` as `SUMMARY | READONLY`. LinkCheck refuses that link (`linkcheck.propReadonly`).

- **Semantics.** Every comment block, when it ends (an annotation such as `@NiagaraType` included), adds to a file-level
  set the slots it NAMES as link targets. A name is code formatting: `` `status` `` or `{@code status}`;
  `{@link …}` tags are removed first. A link keyword is one of the existing comment keywords, "link target", or
  link/links/linked within four words of from/into/to. Within one sentence, each keyword marks one name: the last
  one before a "from" keyword (the target precedes "linked from <source>"), else the first one after it, else the
  nearest one before it. A READONLY property whose name is in the set FAILs as `(class comment)`.
- **Precision kept.** The per-property block keeps its old meaning (a keyword marks its whole run of properties,
  and an annotation still ends it), so the class header "// Facade room panel: link-in display slots" of the
  `comment` fixture still marks nothing. A javadoc with a keyword and no named slot marks none, and a sentence that
  names two slots marks only the one next to its keyword (`setpoint` and `defrostStart` stay unmarked in the
  `classdoc` fixture).
- **Expected new hit.** The rule now FAILs `fixtures/MinimalDash/.../BMinimalDash.java:35` `status` on main. That is
  the defect itself; MinimalDash.status is being fixed on another branch.

## Evidence
- RED before the change: goldens `classdoc-readonly-1` and `classdoc-annotations-1` got `exit 0` and wanted the
  `LTF1: link-in target "status" (class comment)` row; bats LTF-classdoc and LTF-classdoc-named failed at
  `[ "$status" -eq 1 ]` `[ev: go/internal/lint/rules/linktargetflags/testdata/classdoc-readonly-1.txt]`
- Goldens `classdoc-summary-1` (the same with `status` SUMMARY only, exit 0) and `classdoc-unnamed-1` (a keyword
  javadoc that names no slot over two READONLY slots, exit 0) were green before and after
- Mutations: dropping the harvest call fails both bats pins; marking every code name of a keyword block fails
  LTF-classdoc-named `[ev: tests/lint-link-target-flags.bats]`
- Kit scan: `fixtures/**` one new row (MinimalDash `status`, above); `templates/**` exit 0 before and after

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A bare word ("links control points into status") marks nothing, because "status" is also prose. If a real facade names its link-in slots without code formatting, consider matching bare words against the file's declared property names inside a keyword sentence only | `toolbelt/lint-link-target-flags.sh` heuristic limits | `[ev: retro r10-ltf1-marker-reset]` |
| Δ2 | `types/dashboard.md` § rt should tell facade authors to name each link-in slot in code formatting in the class javadoc ({@code status}), so LTF1 sees it without a per-property comment or a wiring map | `types/dashboard.md` § rt — the facade | `[ev: retro r10-ltf1-marker-reset]` |

## Lessons
- A lint that reads a comment block must say what ends the block; in Niagara sources the annotations stand between
  the class javadoc and the properties, so "another annotation ends the block" silently disowned every class javadoc.
