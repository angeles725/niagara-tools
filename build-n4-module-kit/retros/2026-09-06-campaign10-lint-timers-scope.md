<!-- review-status: pending -->
# 2026-09-06 · kit · campaign10-lint-timers-scope

**Session**: Campaign 10 PR1 — S21 companion-flag false positive fix (anyNoHardware on BDefrostController)
**Delta count**: 4

## What happened

`lint-timers.sh` on ColdRoomPan-rt/src emitted `FAIL companion-flag … 'anyNoHardware'` (exit 1) on kit
`df8c7ec`. `anyNoHardware` is a METHOD-LOCAL `boolean` inside `requestDefrostCycle()` (BDefrostController.java:718).
The root cause was twofold: Pass 1's candidate regex `[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*\(` matched
`@NiagaraProperty(` as a method name, then the forward brace-walk swallowed the class body as the
"method body"; within that entire body it found `anyNoHardware = true;` and a `Clock.schedule*` call
(from an unrelated method). The check never distinguished class FIELDs from method LOCALs — any
`boolean X = … ; X = true;` near a schedule was a candidate, regardless of declaration scope.

[ev: kit lint-timers.sh:143-186 @ df8c7ec] [ev: client BDefrostController.java:713,:718,:726 @ ff1b659]
[ev: S21-neg RED bats exit 1 before fix] [ev: S21-smoke RED exit 1 before fix on ColdRoomPan-rt]

## Evidence

- `bats tests/lint-timers.bats` RED: S21-neg (test 13) + S21-smoke (test 15) FAIL on kit df8c7ec. `[ev: RED output 52ebd11]`
- GREEN after fix: all 15 tests pass; full `bats tests/` 385/385. `[ev: GREEN output post-fix]`
- Mutation RED: drop field-scope guard (`prev_depth == 1`) from Phase 1 → anyNoHardware treated as a FIELD → S21-neg re-FAILs with `FAIL companion-flag anyNoHardware`. `[ev: mut scratchpad run 2026-09-06]`
- Real-tree smoke ColdRoomPan-rt: exit 0, anyNoHardware ABSENT; CompPan-rt: exit 0, 1 PASS timer-ticket; DashboardPan-rt: exit 0. `[ev: ff1b659 run 2026-09-06]`
- shellcheck 0.10.0 exit 0; kit-links.bats 8/8. `[ev: SC-9 2026-09-06]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | When a companion-flag lint check is added, the "same method body" scope must use a method-boundary parser (not a forward brace-walk), and the flag must be proven a class FIELD (depth-1 declaration), not just any boolean/int assignment. | METHODOLOGY.md §Pre-gate checks | [ev: retro campaign10-lint-timers-scope] |
| 2 | A candidate regex that matches `identifier(` will hit annotation names like `@NiagaraProperty(` — always scope candidate detection to lines where the net brace change is +1 AND brace_depth after is >= 2 (class body at depth 1 is never a method). | BUILD-LOOP.md §5 toolbelt | [ev: retro campaign10-lint-timers-scope] |
| 3 | Port the section-D method-boundary parser from lint-silent-protection.sh when writing any new per-method scope check in the toolbelt; never hand-roll a forward brace-walk. | METHODOLOGY.md §Kit maintenance | [ev: retro campaign10-lint-timers-scope] |
| 4 | The "field vs local" distinction is load-bearing for companion-flag accuracy: field at brace_depth 1, local at depth >= 2; deriving both from ONE brace-depth counter in ONE pass prevents the walk-disagreement class of bug. | BUILD-LOOP.md §5 toolbelt | [ev: retro campaign10-lint-timers-scope] |

## Lessons

- A lint check that collects identifiers must scope collection to the correct declaration depth; a class-scope FIELD (depth 1) behaves fundamentally differently from a method-local (depth >= 2) across stop/restart cycles. [ev: retro campaign10-lint-timers-scope]
- Forward brace-walks from a candidate line are fragile — they pick up whichever `{` comes next, which may be a class body if the candidate regex fires on an annotation. Port the section-D net-brace-open parser instead. [ev: retro campaign10-lint-timers-scope]
- Comment stripping before awk pattern work prevents annotation strings in comments (e.g. `// like @NiagaraProperty`) from triggering false candidate matches; blank not delete so line numbers are preserved. [ev: retro campaign10-lint-timers-scope]
- The brace_depth >= 2 guard for method-open acceptance is a cheap, exact, proof-by-invariant fix: a top-level class body opens at depth 1 and any real method opens at depth >= 2 — no token blacklist can be exhaustive. [ev: retro campaign10-lint-timers-scope]

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign10-lint-timers-scope.md | kit | 2026-09-06 | pending | 4 |`
