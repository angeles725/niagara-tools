<!-- review-status: folded -->
# Retro: campaign8-lint-delays — lint-delays.sh Clock.schedule delay-floor lint

**Session**: Campaign 8 PR1 · 2026-09-05
**Change**: toolbelt/lint-delays.sh (new), tests/lint-delays.bats (LD1-LD11), BUILD-LOOP.md §5, skill/SKILL.md v0.7→0.8, openspec-design D2b+D9b+D2c
**Delta count**: 4 proposed deltas

## Context

PR1 adds `lint-delays.sh`, a static lint that detects Clock.schedule/schedulePeriodically
call sites where the delay argument has no proven strictly-positive (>0) floor. Grounded in
a PRODUCTION incident: PANCCADIA ColdRoom_3 (2026-09-04) lost defrost silently 5× because
`BDefrostController.armTrigger()` computed `d = Math.max(delayMs, 0L)` — overdue → d=0 →
EngineManager threw "time <= 0" at runtime, the catch swallowed it. Campaign-8 spec B801.

Real smokes: pre-fix tree 4f5f1c7 exits 1 (BDefrostController :556/:566/:620/:664 FAIL;
BEvaporatorUnit :825/:900/:913/:1091 PASS guarded). Fixed tree c66e412 exits 0: BDefrostController
clean (D2b resolves positiveDelayMs as proven floor); BEvaporatorUnit :825/:923/:936/:1123 all
PASS (D2c guard recognition — guarded at :818/:920/:935/:1113).

## Proven Lessons

### Delta 1 — cross-file helper resolution (D2b): resolve static long helpers one level

**Observation**: the production fix routed every delay through `ColdRoomControl.positiveDelayMs(long ms)` —
a `static long` helper in a separate file. A lint that only recognises inline `Math.max` has zero
bite against this pattern: `BRelTime.make(ColdRoomControl.positiveDelayMs(getInterval().getMillis()))`
looks like an unresolved call to the naive scanner.

**Lesson**: whenever a delay-floor lint sees `BRelTime.make(<call>)` or a backward variable
assignment `varname = <Class>.<fn>(...)`, it must resolve the helper one level. A pre-pass awk
collects all `static long <fn>(` bodies across all *.java files (dot-dirs excluded) and marks
a function as a proven strictly-positive floor if its body contains `Math.max(…,N≥1)`, a `>=1`
comparison, a `<1?1:x` shape, or `return 1L`. The lookup is done by function name only (no class
disambiguation needed for a proof of floor). `FNR==1` resets awk state across files so method
bodies never bleed between files.

**Caveat**: the lookup matches by function name only; a same-named non-floor helper in a different
class would be incorrectly accepted. Acceptable for the kit's domain (internal module code, small
surface, no dependency injection). A false negative is a missed FAIL, not a crash.

### Delta 2 — dot-dir exclusion (D9b): `.deploy-baseline/` lives inside the artifact

**Observation**: `<artifact>/.deploy-baseline/` contains baseline snapshots for schema-risk.sh.
A scanner that walks all subdirs would double-count baseline *.java files and produce spurious
FAILs on the snapshot copies.

**Lesson**: every file-walking toolbelt script MUST prune dot-directories using
`find … -type d -name '.*' -prune -o … -print`. The exclusion is already in lint-delays.sh
from the initial implementation. Verified with a `BadDelay.java` planted under
`<tmp>/.deploy-baseline/` → exit 0, no rows. The same prune applies to every PR1-PR8 scanner
(lint-timers in PR3, verify-module/slot-coverage in PR4/PR5, schema-risk in PR8).

### Delta 3 — stale-status lesson: a worker that goes silent mid-apply must leave the worktree resumable

**Observation**: the previous C8 PR1 worker was stopped 36 min into apply. On resume, the
worktree had lint-delays.sh UNTRACKED and bats GREEN 9/9, but no commit, no routing, and the
design addendum D9b was unstaged. The lead's state description was sufficient to resume without
re-reading the whole scope.

**Lesson**: a finisher must accept a mid-apply worktree and continue from the lead-provided state
rather than restart. Workers should commit incrementally (WIP commit after GREEN) so a stop leaves
the worktree cleanly resumable with `git stash` or just a commit of the current work.

### Delta 4 — same-method guard recognition (D2c): a positivity guard on the same expression proves the floor

**Observation**: BEvaporatorUnit (fixed tree c66e412) has 4 Clock.schedule calls that are
each preceded by a same-method positivity guard on the SAME delay expression: `:818` guards
`delayMs > 0L` before `:825`; `:920` guards `getStartDelay().getMillis() == 0L` (zero-branch
takes the else) before `:923`; `:935` guards `getStopDelay().getMillis() > 0L` before `:936`;
`:1113` guards `getFanOffDelay().getMillis() == 0L` (zero handled) before `:1123`. The first
lint cut FAILed all four as "facet-min-zero" — true false positives. A guard on a DIFFERENT
expression (MisGuarded: `getOther()` guards the schedule for `getStartDelay()`) does NOT count.

**Lesson**: a delay-floor lint must recognize within-method positivity guards on the EXACT
same delay expression. Guard forms accepted: `<expr> > 0`, `<expr> >= 1`, `<expr>.getMillis() > 0`,
`<expr>().getMillis() > 0`, `<expr>().getMillis() >= 1`, or `<expr>().getMillis() == 0` (zero-branch
does not reach the schedule). Detection is backward-scanning from the schedule line (up to 80 lines).
Single-line comments are stripped before pattern matching — otherwise comment text like
`// delayMs > 0L` would false-positively trigger the guard and mask a missing guard. The guard
lookup is EXPRESSION-SPECIFIC: `find_guard(lnum, expr)` checks `index(line, expr)` first, so
a guard on `getOther` never covers a schedule on `getStartDelay`.
